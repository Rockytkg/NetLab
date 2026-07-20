// Package radius 提供 RADIUS 认证计费的管理业务逻辑（CRUD、会话管理、统计）。
package radius

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"netlab-backend/config"
	dtorequest "netlab-backend/internal/dto/request"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

// SecretMask 是更新请求中"密钥不变"的占位值：
// 前端编辑表单不回显真实密钥，提交该掩码表示保留原值。
const SecretMask = "__UNCHANGED__"

// Service 是 RADIUS 管理业务服务。
type Service struct {
	users      *repository.RadiusUserRepository
	nas        *repository.RadiusNasRepository
	sessions   *repository.RadiusSessionRepository
	accounting *repository.RadiusAccountingRepository
	authLogs   *repository.RadiusAuthLogRepository
	certs      *repository.RadiusCertRepository
	bypass     *repository.RadiusBypassRepository
	cipher     *crypto.AESCipher
	// manager 持有 RADIUS 运行时生命周期；其 CoA() 在服务未运行时返回 nil。
	manager *radiusd.Manager
	// cfgSvc 是管理端 DB 配置（radius.* blob）的存取服务。
	cfgSvc *sysconfig.Service
	// envCfg 是进程启动时从环境变量加载的 RADIUS 配置，作为生效配置的底座。
	envCfg config.RadiusConfig
}

// NewService 构造 RADIUS 管理服务。
func NewService(
	users *repository.RadiusUserRepository,
	nas *repository.RadiusNasRepository,
	sessions *repository.RadiusSessionRepository,
	accounting *repository.RadiusAccountingRepository,
	authLogs *repository.RadiusAuthLogRepository,
	certs *repository.RadiusCertRepository,
	bypass *repository.RadiusBypassRepository,
	cipher *crypto.AESCipher,
	manager *radiusd.Manager,
	cfgSvc *sysconfig.Service,
	envCfg config.RadiusConfig,
) *Service {
	return &Service{
		users: users, nas: nas, sessions: sessions, accounting: accounting,
		authLogs: authLogs, certs: certs, bypass: bypass, cipher: cipher,
		manager: manager, cfgSvc: cfgSvc, envCfg: envCfg,
	}
}

// —— 认证用户 ——

// ListUsers 分页查询认证用户（回填套餐名与在线数；分组批量查询避免 N+1，
// 回填失败时仅放弃回填不阻断列表，与既有行为一致）。
func (s *Service) ListUsers(ctx context.Context, page, size int, keyword, status string) ([]model.RadiusUser, int64, *apperrors.AppError) {
	users, total, err := s.users.List(ctx, page, size, keyword, status)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list radius users", err)
	}

	profileIDs := make([]uint64, 0, len(users))
	usernames := make([]string, 0, len(users))
	seenProfiles := make(map[uint64]struct{}, len(users))
	for i := range users {
		usernames = append(usernames, users[i].Username)
		if users[i].ProfileID != nil {
			if _, ok := seenProfiles[*users[i].ProfileID]; !ok {
				seenProfiles[*users[i].ProfileID] = struct{}{}
				profileIDs = append(profileIDs, *users[i].ProfileID)
			}
		}
	}
	profiles, perr := s.users.GetProfilesByIDs(ctx, profileIDs)
	if perr != nil {
		profiles = nil
	}
	onlineCounts, cerr := s.sessions.CountByUsernames(ctx, usernames)
	if cerr != nil {
		onlineCounts = nil
	}
	for i := range users {
		if users[i].ProfileID != nil {
			// 套餐缺失时保持 nil（仅使用户本行配置，不阻断列表）。
			users[i].Profile = profiles[*users[i].ProfileID]
		}
		users[i].OnlineCount = int(onlineCounts[users[i].Username])
	}
	return users, total, nil
}

// CreateUser 创建认证用户（密码加密存储）。
func (s *Service) CreateUser(ctx context.Context, req *dtorequest.RadiusUserUpsertRequest) (*model.RadiusUser, *apperrors.AppError) {
	if req.Password == "" {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "password is required")
	}
	if existing, err := s.users.GetByUsername(ctx, req.Username); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check username", err)
	} else if existing != nil {
		return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "username already exists")
	}
	if appErr := s.checkProfileExists(ctx, req.ProfileID); appErr != nil {
		return nil, appErr
	}

	encrypted, err := s.cipher.Encrypt(req.Password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt password", err)
	}

	// 过期时间留空时默认一年后过期。
	expireTime := time.Now().AddDate(1, 0, 0)
	if req.ExpireTime != nil {
		expireTime = *req.ExpireTime
	}

	user := &model.RadiusUser{
		Username:                req.Username,
		Password:                encrypted,
		ProfileID:               req.ProfileID,
		ProfileLinkMode:         intValue(req.ProfileLinkMode, model.RadiusLinkModeDynamic),
		Realname:                req.Realname,
		Email:                   req.Email,
		Mobile:                  req.Mobile,
		Address:                 req.Address,
		MacAddr:                 model.NormalizeMacList(req.MacAddr),
		Vlanid1:                 intValue(req.Vlanid1, 0),
		Vlanid2:                 intValue(req.Vlanid2, 0),
		BindMac:                 boolValue(req.BindMac),
		BindVlan:                boolValue(req.BindVlan),
		IpAddr:                  req.IpAddr,
		IpV6Addr:                req.IpV6Addr,
		AddrPool:                req.AddrPool,
		IPv6PrefixPool:          req.IPv6PrefixPool,
		DelegatedIpv6Prefix:     req.DelegatedIpv6Prefix,
		DelegatedIpv6PrefixPool: req.DelegatedIpv6PrefixPool,
		ActiveNum:               intValue(req.ActiveNum, 0),
		UpRate:                  intValue(req.UpRate, 0),
		DownRate:                intValue(req.DownRate, 0),
		Domain:                  req.Domain,
		ExpireTime:              expireTime,
		Status:                  statusOrDefault(req.Status),
		Remark:                  req.Remark,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create radius user", err)
	}
	return user, nil
}

// UpdateUser 更新认证用户；Password 为空或为掩码时保留原密码。
func (s *Service) UpdateUser(ctx context.Context, id uint64, req *dtorequest.RadiusUserUpsertRequest) (*model.RadiusUser, *apperrors.AppError) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius user", err)
	}
	if user == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "radius user not found")
	}
	if user.Username != req.Username {
		if existing, err := s.users.GetByUsername(ctx, req.Username); err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check username", err)
		} else if existing != nil {
			return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "username already exists")
		}
	}
	if appErr := s.checkProfileExists(ctx, req.ProfileID); appErr != nil {
		return nil, appErr
	}

	user.Username = req.Username
	if req.Password != "" && req.Password != SecretMask {
		encrypted, err := s.cipher.Encrypt(req.Password)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt password", err)
		}
		user.Password = encrypted
	}
	user.ProfileID = req.ProfileID
	user.ProfileLinkMode = intValue(req.ProfileLinkMode, model.RadiusLinkModeDynamic)
	user.Realname = req.Realname
	user.Email = req.Email
	user.Mobile = req.Mobile
	user.Address = req.Address
	user.MacAddr = model.NormalizeMacList(req.MacAddr)
	user.Vlanid1 = intValue(req.Vlanid1, 0)
	user.Vlanid2 = intValue(req.Vlanid2, 0)
	user.BindMac = boolValue(req.BindMac)
	user.BindVlan = boolValue(req.BindVlan)
	user.IpAddr = req.IpAddr
	user.IpV6Addr = req.IpV6Addr
	user.AddrPool = req.AddrPool
	user.IPv6PrefixPool = req.IPv6PrefixPool
	user.DelegatedIpv6Prefix = req.DelegatedIpv6Prefix
	user.DelegatedIpv6PrefixPool = req.DelegatedIpv6PrefixPool
	user.ActiveNum = intValue(req.ActiveNum, 0)
	user.UpRate = intValue(req.UpRate, 0)
	user.DownRate = intValue(req.DownRate, 0)
	user.Domain = req.Domain
	// 过期时间留空时保留原值。
	if req.ExpireTime != nil {
		user.ExpireTime = *req.ExpireTime
	}
	user.Status = statusOrDefault(req.Status)
	user.Remark = req.Remark

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update radius user", err)
	}
	return user, nil
}

// DeleteUser 删除认证用户，并尽力清理其在线会话（清理失败仅记录日志，不阻断删除）。
func (s *Service) DeleteUser(ctx context.Context, id uint64) *apperrors.AppError {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius user", err)
	}
	if user == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "radius user not found")
	}
	if err := s.users.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete radius user", err)
	}
	if err := s.sessions.DeleteByUsername(ctx, user.Username); err != nil {
		log.Printf("[RADIUS] delete online sessions for removed user %s failed: %v", user.Username, err)
	}
	return nil
}

// —— 策略套餐 ——

// ListProfiles 分页查询套餐（分组统计回填引用用户数，避免逐行 N+1；
// 回填失败时仅放弃回填不阻断列表）。
func (s *Service) ListProfiles(ctx context.Context, page, size int, keyword string) ([]model.RadiusProfile, int64, *apperrors.AppError) {
	profiles, total, err := s.users.ListProfiles(ctx, page, size, keyword)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list radius profiles", err)
	}
	ids := make([]uint64, 0, len(profiles))
	for i := range profiles {
		ids = append(ids, profiles[i].ID)
	}
	if counts, cerr := s.users.CountByProfileIDs(ctx, ids); cerr == nil {
		for i := range profiles {
			profiles[i].UserCount = counts[profiles[i].ID]
		}
	}
	return profiles, total, nil
}

// ListAllProfiles 返回全部套餐（下拉选项用）。
func (s *Service) ListAllProfiles(ctx context.Context) ([]model.RadiusProfile, *apperrors.AppError) {
	profiles, err := s.users.ListAllProfiles(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list radius profiles", err)
	}
	return profiles, nil
}

// CountProfileUsers 统计套餐引用用户数。
func (s *Service) CountProfileUsers(ctx context.Context, profileID uint64) int64 {
	count, err := s.users.CountByProfileID(ctx, profileID)
	if err != nil {
		return 0
	}
	return count
}

// CreateProfile 创建套餐。
func (s *Service) CreateProfile(ctx context.Context, req *dtorequest.RadiusProfileUpsertRequest) (*model.RadiusProfile, *apperrors.AppError) {
	if existing, err := s.users.GetProfileByName(ctx, req.Name); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check profile name", err)
	} else if existing != nil {
		return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "profile name already exists")
	}
	profile := &model.RadiusProfile{
		Name:                    req.Name,
		Status:                  statusOrDefault(req.Status),
		AddrPool:                req.AddrPool,
		IPv6PrefixPool:          req.IPv6PrefixPool,
		DelegatedIpv6PrefixPool: req.DelegatedIpv6PrefixPool,
		ActiveNum:               intValue(req.ActiveNum, 0),
		UpRate:                  intValue(req.UpRate, 0),
		DownRate:                intValue(req.DownRate, 0),
		Domain:                  req.Domain,
		BindMac:                 boolValue(req.BindMac),
		BindVlan:                boolValue(req.BindVlan),
		Remark:                  req.Remark,
	}
	if err := s.users.CreateProfile(ctx, profile); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create radius profile", err)
	}
	return profile, nil
}

// UpdateProfile 更新套餐。
func (s *Service) UpdateProfile(ctx context.Context, id uint64, req *dtorequest.RadiusProfileUpsertRequest) (*model.RadiusProfile, *apperrors.AppError) {
	profile, err := s.users.GetProfileByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius profile", err)
	}
	if profile == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "radius profile not found")
	}
	if profile.Name != req.Name {
		if existing, err := s.users.GetProfileByName(ctx, req.Name); err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check profile name", err)
		} else if existing != nil {
			return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "profile name already exists")
		}
	}
	profile.Name = req.Name
	profile.Status = statusOrDefault(req.Status)
	profile.AddrPool = req.AddrPool
	profile.IPv6PrefixPool = req.IPv6PrefixPool
	profile.DelegatedIpv6PrefixPool = req.DelegatedIpv6PrefixPool
	profile.ActiveNum = intValue(req.ActiveNum, 0)
	profile.UpRate = intValue(req.UpRate, 0)
	profile.DownRate = intValue(req.DownRate, 0)
	profile.Domain = req.Domain
	profile.BindMac = boolValue(req.BindMac)
	profile.BindVlan = boolValue(req.BindVlan)
	profile.Remark = req.Remark
	if err := s.users.UpdateProfile(ctx, profile); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update radius profile", err)
	}
	return profile, nil
}

// DeleteProfile 删除套餐；被用户或终端准入规则引用时禁止删除。
func (s *Service) DeleteProfile(ctx context.Context, id uint64) *apperrors.AppError {
	profile, err := s.users.GetProfileByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius profile", err)
	}
	if profile == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "radius profile not found")
	}
	if count := s.CountProfileUsers(ctx, id); count > 0 {
		return apperrors.New(apperrors.ErrCodeResourceInUse, "profile is referenced by users")
	}
	if count, err := s.bypass.CountByProfileID(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to count bypass profile references", err)
	} else if count > 0 {
		return apperrors.New(apperrors.ErrCodeResourceInUse, "profile is referenced by terminal access rules")
	}
	if err := s.users.DeleteProfile(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete radius profile", err)
	}
	return nil
}

// —— NAS 设备 ——

// ListNas 分页查询 NAS 设备。
func (s *Service) ListNas(ctx context.Context, page, size int, keyword string) ([]model.RadiusNas, int64, *apperrors.AppError) {
	items, total, err := s.nas.List(ctx, page, size, keyword)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list nas devices", err)
	}
	return items, total, nil
}

// CreateNas 创建 NAS 设备（密钥加密存储）。
func (s *Service) CreateNas(ctx context.Context, req *dtorequest.RadiusNasUpsertRequest) (*model.RadiusNas, *apperrors.AppError) {
	if req.Secret == "" {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "secret is required")
	}
	encrypted, err := s.cipher.Encrypt(req.Secret)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt secret", err)
	}
	nas := &model.RadiusNas{
		Name:       req.Name,
		Identifier: req.Identifier,
		Hostname:   req.Hostname,
		Ipaddr:     req.Ipaddr,
		Secret:     encrypted,
		CoaPort:    intValue(req.CoaPort, radiusd.DefaultCoAPort),
		Model:      req.Model,
		VendorCode: req.VendorCode,
		Status:     statusOrDefault(req.Status),
		Tags:       req.Tags,
		Remark:     req.Remark,
	}
	if err := s.nas.Create(ctx, nas); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create nas device", err)
	}
	return nas, nil
}

// UpdateNas 更新 NAS 设备；Secret 为空或为掩码时保留原密钥。
func (s *Service) UpdateNas(ctx context.Context, id uint64, req *dtorequest.RadiusNasUpsertRequest) (*model.RadiusNas, *apperrors.AppError) {
	nas, err := s.nas.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load nas device", err)
	}
	if nas == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "nas device not found")
	}
	nas.Name = req.Name
	nas.Identifier = req.Identifier
	nas.Hostname = req.Hostname
	nas.Ipaddr = req.Ipaddr
	if req.Secret != "" && req.Secret != SecretMask {
		encrypted, err := s.cipher.Encrypt(req.Secret)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt secret", err)
		}
		nas.Secret = encrypted
	}
	nas.CoaPort = intValue(req.CoaPort, radiusd.DefaultCoAPort)
	nas.Model = req.Model
	nas.VendorCode = req.VendorCode
	nas.Status = statusOrDefault(req.Status)
	nas.Tags = req.Tags
	nas.Remark = req.Remark
	if err := s.nas.Update(ctx, nas); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update nas device", err)
	}
	return nas, nil
}

// DeleteNas 删除 NAS 设备。
func (s *Service) DeleteNas(ctx context.Context, id uint64) *apperrors.AppError {
	nas, err := s.nas.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load nas device", err)
	}
	if nas == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "nas device not found")
	}
	if err := s.nas.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete nas device", err)
	}
	return nil
}

// —— 在线会话 ——

// ListSessions 分页查询在线会话。
func (s *Service) ListSessions(ctx context.Context, page, size int, username, nasAddr, macAddr string) ([]model.RadiusOnline, int64, *apperrors.AppError) {
	items, total, err := s.sessions.List(ctx, page, size, username, nasAddr, macAddr)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list online sessions", err)
	}
	return items, total, nil
}

// KickSession 通过 CoA/DM 将在线会话踢下线；成功后删除本地在线记录。
func (s *Service) KickSession(ctx context.Context, id uint64) (*radiusd.CoAResult, *apperrors.AppError) {
	coa := s.manager.CoA()
	if coa == nil {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "radius service is not enabled")
	}
	session, err := s.sessions.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load online session", err)
	}
	if session == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "online session not found")
	}
	result, err := coa.DisconnectSession(ctx, session.AcctSessionId)
	if err != nil {
		if errors.Is(err, radiusd.ErrSessionNotFound) {
			return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "online session not found")
		}
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to disconnect session", err)
	}
	if result.Success {
		_ = s.sessions.Delete(ctx, session.AcctSessionId)
	}
	return result, nil
}

// ModifySession 通过 CoA-Request 向在线会话下发授权变更（会话超时/过滤器）。
// CoA 不是下线操作，成功后保留本地在线记录。
func (s *Service) ModifySession(ctx context.Context, id uint64, sessionTimeout *int, filterID string) (*radiusd.CoAResult, *apperrors.AppError) {
	coa := s.manager.CoA()
	if coa == nil {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "radius service is not enabled")
	}
	session, err := s.sessions.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load online session", err)
	}
	if session == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "online session not found")
	}

	// 组装授权变更属性：Session-Timeout（秒）与 Filter-Id（≤253 字节）。
	var setters []radiusd.CoAAttributeSetter
	if sessionTimeout != nil && *sessionTimeout > 0 {
		setters = append(setters, radiusd.WithSessionTimeout(uint32(*sessionTimeout)))
	}
	if filterID != "" && len(filterID) <= 253 {
		setters = append(setters, radiusd.WithFilterID(filterID))
	}
	if len(setters) == 0 {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "sessionTimeout 与 filterId 至少一项有效")
	}

	result, err := coa.CoASession(ctx, session.AcctSessionId, setters...)
	if err != nil {
		if errors.Is(err, radiusd.ErrSessionNotFound) {
			return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "online session not found")
		}
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to send coa request", err)
	}
	return result, nil
}

// —— 记账记录 ——

// ListAccounting 分页查询记账记录。
func (s *Service) ListAccounting(ctx context.Context, page, size int, username string, startTime, endTime *time.Time) ([]model.RadiusAccounting, int64, *apperrors.AppError) {
	items, total, err := s.accounting.List(ctx, page, size, username, startTime, endTime)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list accounting records", err)
	}
	return items, total, nil
}

// —— 认证日志 ——

// ListAuthLogs 分页查询认证日志。
func (s *Service) ListAuthLogs(ctx context.Context, page, size int, keyword, result string) ([]model.RadiusAuthLog, int64, *apperrors.AppError) {
	items, total, err := s.authLogs.List(ctx, page, size, keyword, result)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list auth logs", err)
	}
	return items, total, nil
}

// DeleteAuthLogs 批量删除认证日志。
func (s *Service) DeleteAuthLogs(ctx context.Context, ids []uint64) (int64, *apperrors.AppError) {
	deleted, err := s.authLogs.Delete(ctx, ids)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete auth logs", err)
	}
	return deleted, nil
}

// —— 免认证规则 ——

// ListBypassRules 分页查询免认证规则。
func (s *Service) ListBypassRules(ctx context.Context, page, size int, keyword string) ([]model.RadiusBypass, int64, *apperrors.AppError) {
	items, total, err := s.bypass.List(ctx, page, size, keyword)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list bypass rules", err)
	}
	return items, total, nil
}

// CreateBypassRule 创建免认证规则（类型与取值校验后归一化存储）。
func (s *Service) CreateBypassRule(ctx context.Context, req *dtorequest.RadiusBypassUpsertRequest) (*model.RadiusBypass, *apperrors.AppError) {
	value, appErr := normalizeBypassValue(req.Type, req.Value)
	if appErr != nil {
		return nil, appErr
	}
	if appErr := s.checkBypassProfile(ctx, req.ProfileID); appErr != nil {
		return nil, appErr
	}
	if req.Type == model.RadiusBypassTypeIP && req.NasID == nil {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "ip bypass rule requires a nas scope")
	}
	if appErr := s.checkBypassNAS(ctx, req.NasID); appErr != nil {
		return nil, appErr
	}
	if req.ExpireTime != nil && !req.ExpireTime.After(time.Now()) {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "bypass rule expiry must be in the future")
	}
	if existing, err := s.bypass.GetByTypeValue(ctx, req.Type, value); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check bypass rule", err)
	} else if existing != nil {
		return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "bypass rule already exists")
	}
	rule := &model.RadiusBypass{
		Type:       req.Type,
		Value:      value,
		ProfileID:  req.ProfileID,
		NasID:      req.NasID,
		ExpireTime: req.ExpireTime,
		Status:     statusOrDefault(req.Status),
		Remark:     req.Remark,
	}
	if err := s.bypass.Create(ctx, rule); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create bypass rule", err)
	}
	s.manager.InvalidateBypassRules()
	return rule, nil
}

// UpdateBypassRule 更新免认证规则；(type, value) 冲突时返回重复错误。
func (s *Service) UpdateBypassRule(ctx context.Context, id uint64, req *dtorequest.RadiusBypassUpsertRequest) (*model.RadiusBypass, *apperrors.AppError) {
	rule, err := s.bypass.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load bypass rule", err)
	}
	if rule == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "bypass rule not found")
	}
	value, appErr := normalizeBypassValue(req.Type, req.Value)
	if appErr != nil {
		return nil, appErr
	}
	if appErr := s.checkBypassProfile(ctx, req.ProfileID); appErr != nil {
		return nil, appErr
	}
	if req.Type == model.RadiusBypassTypeIP && req.NasID == nil {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "ip bypass rule requires a nas scope")
	}
	if appErr := s.checkBypassNAS(ctx, req.NasID); appErr != nil {
		return nil, appErr
	}
	if req.ExpireTime != nil && !req.ExpireTime.After(time.Now()) {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "bypass rule expiry must be in the future")
	}
	if rule.Type != req.Type || rule.Value != value {
		if existing, err := s.bypass.GetByTypeValue(ctx, req.Type, value); err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check bypass rule", err)
		} else if existing != nil && existing.ID != id {
			return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "bypass rule already exists")
		}
	}
	rule.Type = req.Type
	rule.Value = value
	rule.ProfileID = req.ProfileID
	rule.NasID = req.NasID
	rule.ExpireTime = req.ExpireTime
	rule.Status = statusOrDefault(req.Status)
	rule.Remark = req.Remark
	if err := s.bypass.Update(ctx, rule); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update bypass rule", err)
	}
	s.manager.InvalidateBypassRules()
	return rule, nil
}

// DeleteBypassRule 删除免认证规则。
func (s *Service) DeleteBypassRule(ctx context.Context, id uint64) *apperrors.AppError {
	rule, err := s.bypass.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load bypass rule", err)
	}
	if rule == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "bypass rule not found")
	}
	if err := s.bypass.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete bypass rule", err)
	}
	s.manager.InvalidateBypassRules()
	return nil
}

// normalizeBypassValue 校验并归一化终端准入规则取值。IP 规则只允许单地址，
// 不允许 CIDR，避免一条误配规则放开整段网络。
func normalizeBypassValue(ruleType, value string) (string, *apperrors.AppError) {
	value = strings.TrimSpace(value)
	switch ruleType {
	case model.RadiusBypassTypeMac:
		normalized := model.NormalizeMacList(value)
		if normalized == "" || strings.Contains(normalized, ",") {
			return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "mac bypass rule requires exactly one mac address")
		}
		if _, err := net.ParseMAC(normalized); err != nil {
			return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid mac address")
		}
		return normalized, nil
	case model.RadiusBypassTypeIP:
		ip := net.ParseIP(value)
		if ip == nil || ip.To4() == nil {
			return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "ip bypass rule requires a valid ipv4 address")
		}
		return ip.To4().String(), nil
	default:
		return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "bypass type must be mac or ip")
	}
}

func (s *Service) checkBypassProfile(ctx context.Context, profileID uint64) *apperrors.AppError {
	profile, err := s.users.GetProfileByID(ctx, profileID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load bypass profile", err)
	}
	if profile == nil || profile.Status != model.RadiusUserStatusEnabled {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "bypass profile must be enabled")
	}
	return nil
}

func (s *Service) checkBypassNAS(ctx context.Context, nasID *uint64) *apperrors.AppError {
	if nasID == nil {
		return nil
	}
	nas, err := s.nas.GetByID(ctx, *nasID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load bypass nas", err)
	}
	if nas == nil || nas.Status != model.RadiusNasStatusEnabled {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "bypass nas must be enabled")
	}
	return nil
}

// —— 内部辅助 ——

// checkProfileExists 校验引用的套餐存在。
func (s *Service) checkProfileExists(ctx context.Context, profileID *uint64) *apperrors.AppError {
	if profileID == nil {
		return nil
	}
	profile, err := s.users.GetProfileByID(ctx, *profileID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius profile", err)
	}
	if profile == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "radius profile not found")
	}
	return nil
}

// intValue 解引用 int 指针，nil 时返回 def。
func intValue(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

// boolValue 解引用 bool 指针，nil 时返回 false。
func boolValue(v *bool) bool {
	return v != nil && *v
}

// statusOrDefault 归一化状态取值，空值默认 enabled。
func statusOrDefault(status string) string {
	if status == "" {
		return "enabled"
	}
	return status
}
