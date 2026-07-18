// Package log 提供登录日志的记录与查询能力。
// 记录采用异步写入，绝不影响登录主流程。
package log

import (
	"context"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
)

// 字段长度上限，与 model.LoginLog 的列定义保持一致。
const (
	maxUsernameLen    = 64
	maxIPLen          = 45
	maxUserAgentLen   = 512
	maxFingerprintLen = 128
	maxOSLen          = 64
	maxBrowserLen     = 64
)

// maxDeleteIDs 是单次批量删除的 ID 数量上限。
const maxDeleteIDs = 500

// RoleLeveller 提供按角色 ID 查询管理级别的能力，由 RBAC 服务实现。
type RoleLeveller interface {
	RoleLevel(ctx context.Context, roleID string) (int, error)
}

// LoginLogEntry 是一条待记录的登录日志。
type LoginLogEntry struct {
	UserID      *uint64
	Username    string
	LoginType   string
	Status      string
	IP          string
	UserAgent   string
	Fingerprint string
	OS          string
	Browser     string
}

// Service 提供登录日志的异步记录与分级查询。
type Service struct {
	loginLogRepo *repository.LoginLogRepository
	userRepo     *repository.UserRepository
	roleLeveller RoleLeveller
	logger       *zap.Logger
}

// NewService 创建一个新的登录日志 Service。
func NewService(loginLogRepo *repository.LoginLogRepository, userRepo *repository.UserRepository, roleLeveller RoleLeveller, logger *zap.Logger) *Service {
	return &Service{
		loginLogRepo: loginLogRepo,
		userRepo:     userRepo,
		roleLeveller: roleLeveller,
		logger:       logger,
	}
}

// Record 异步写入一条登录日志。
// 字段截断在调用方 goroutine 内同步完成，写入失败仅记录告警，
// 永不返回错误、绝不 panic 到调用方。
func (s *Service) Record(entry LoginLogEntry) {
	entry.Username = truncate(entry.Username, maxUsernameLen)
	entry.IP = truncate(entry.IP, maxIPLen)
	entry.UserAgent = truncate(entry.UserAgent, maxUserAgentLen)
	entry.Fingerprint = truncate(entry.Fingerprint, maxFingerprintLen)
	entry.OS = truncate(entry.OS, maxOSLen)
	entry.Browser = truncate(entry.Browser, maxBrowserLen)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warn("login log: panic recovered", zap.Any("panic", r))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 失败场景常常拿不到用户 ID，按用户名补查以便按角色级别过滤可见性；
		// 查不到（如暴力破解不存在的账号）保持 NULL。
		if entry.UserID == nil && entry.Username != "" {
			user, err := s.userRepo.FindByUsernameOrEmail(ctx, entry.Username)
			if err != nil {
				s.logger.Warn("login log: backfill user failed", zap.String("username", entry.Username), zap.Error(err))
			} else if user != nil {
				uid := user.ID
				entry.UserID = &uid
			}
		}

		if err := s.loginLogRepo.Create(ctx, &model.LoginLog{
			UserID:      entry.UserID,
			Username:    entry.Username,
			LoginType:   entry.LoginType,
			Status:      entry.Status,
			IP:          entry.IP,
			UserAgent:   entry.UserAgent,
			Fingerprint: entry.Fingerprint,
			OS:          entry.OS,
			Browser:     entry.Browser,
		}); err != nil {
			s.logger.Warn("login log: create failed", zap.String("username", entry.Username), zap.Error(err))
		}
	}()
}

// List 按操作者角色的管理级别分页查询登录日志：
// 仅返回目标用户管理级别不超过操作者的记录。
func (s *Service) List(ctx context.Context, actorRoleID string, page, size int, keyword, status, loginType string) ([]model.LoginLog, int64, *apperrors.AppError) {
	level, err := s.roleLeveller.RoleLevel(ctx, actorRoleID)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to resolve actor role level", err)
	}
	logs, total, err := s.loginLogRepo.List(ctx, level, page, size, keyword, status, loginType)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list login logs", err)
	}
	return logs, total, nil
}

// Delete 按 ID 批量删除登录日志，仅删除目标用户管理级别不超过
// 操作者的记录（与 List 的可见性规则一致），返回实际删除条数。
func (s *Service) Delete(ctx context.Context, actorRoleID string, ids []uint64) (int64, *apperrors.AppError) {
	if len(ids) == 0 {
		return 0, apperrors.New(apperrors.ErrCodeInvalidRequest, "ids required")
	}
	if len(ids) > maxDeleteIDs {
		return 0, apperrors.New(apperrors.ErrCodeInvalidRequest, "too many ids")
	}
	level, err := s.roleLeveller.RoleLevel(ctx, actorRoleID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to resolve actor role level", err)
	}
	deleted, err := s.loginLogRepo.Delete(ctx, level, ids)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete login logs", err)
	}
	return deleted, nil
}

// truncate 将 s 截断到 max 个字节以内。
func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
