package auth

import (
	"context"

	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/apperrors"
)

// AdminService 承载系统设置管理的业务逻辑。它是 handler 与运行时配置
// 服务之间的编排层：读取时对密钥字段掩码，写入时委托 ConfigService
// 完成加密与热失效。
type AdminService struct {
	configService  *sysconfig.Service
	passkeyService *PasskeyService
}

// NewAdminService 创建一个新的 AdminService。
func NewAdminService(configService *sysconfig.Service, passkeyService *PasskeyService) *AdminService {
	return &AdminService{
		configService:  configService,
		passkeyService: passkeyService,
	}
}

// AdminOAuthProvider 是返回给管理端的单个 OAuth 提供商配置视图。
type AdminOAuthProvider struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"` // 掩码
	RedirectURL  string `json:"redirectUrl"`
	Configured   bool   `json:"configured"`
}

// AdminSettings 是完整的系统设置快照（密钥字段已掩码）。
type AdminSettings struct {
	Security SecurityView            `json:"security"`
	Beian    sysconfig.BeianSettings `json:"beian"`
	SMTP     SMTPView                `json:"smtp"`
	OAuth    []AdminOAuthProvider    `json:"oauth"`
}

// SecurityView 镜像安全策略设置。
type SecurityView = sysconfig.SecuritySettings

// SMTPView 是 SMTP 设置的对外视图，密码字段被掩码。
type SMTPView struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 掩码
	From     string `json:"from"`
	UseTLS   bool   `json:"useTls"`
}

// GetSettings 返回完整的系统设置（密钥掩码）。
func (s *AdminService) GetSettings(ctx context.Context) (*AdminSettings, *apperrors.AppError) {
	sec, err := s.configService.Security(ctx)
	if err != nil {
		return nil, wrapCfg(err)
	}
	beian, err := s.configService.Beian(ctx)
	if err != nil {
		return nil, wrapCfg(err)
	}
	smtp, err := s.configService.SMTP(ctx)
	if err != nil {
		return nil, wrapCfg(err)
	}

	out := &AdminSettings{
		Security: sec,
		Beian:    beian,
		SMTP: SMTPView{
			Enabled:  smtp.Enabled,
			Host:     smtp.Host,
			Port:     smtp.Port,
			Username: smtp.Username,
			Password: maskSecret(smtp.Password),
			From:     smtp.From,
			UseTLS:   smtp.UseTLS,
		},
	}

	for _, id := range sysconfig.ProviderIDs() {
		p, _, err := s.configService.OAuthProvider(ctx, id)
		if err != nil {
			return nil, wrapCfg(err)
		}
		out.OAuth = append(out.OAuth, AdminOAuthProvider{
			ID:           id,
			Name:         sysconfig.ProviderName(id),
			Enabled:      p.Enabled,
			ClientID:     p.ClientID,
			ClientSecret: maskSecret(p.ClientSecret),
			RedirectURL:  p.RedirectURL,
			Configured:   p.ClientID != "" && p.ClientSecret != "",
		})
	}
	return out, nil
}

// UpdateSecurity 更新安全策略。
func (s *AdminService) UpdateSecurity(ctx context.Context, in sysconfig.SecuritySettings) *apperrors.AppError {
	if err := s.configService.SetSecurity(ctx, in); err != nil {
		return wrapCfg(err)
	}
	return nil
}

// UpdateBeian 更新备案信息。
func (s *AdminService) UpdateBeian(ctx context.Context, in sysconfig.BeianSettings) *apperrors.AppError {
	if err := s.configService.SetBeian(ctx, in); err != nil {
		return wrapCfg(err)
	}
	return nil
}

// UpdateSMTP 更新 SMTP 配置。密码字段留空或为掩码时保留原值。
func (s *AdminService) UpdateSMTP(ctx context.Context, in sysconfig.SMTPSettings) *apperrors.AppError {
	if err := s.configService.SetSMTP(ctx, in); err != nil {
		return wrapCfg(err)
	}
	return nil
}

// UpdateOAuthProvider 更新指定 OAuth 提供商配置。
func (s *AdminService) UpdateOAuthProvider(ctx context.Context, id string, in sysconfig.ProviderSettings) *apperrors.AppError {
	_, ok, err := s.configService.OAuthProvider(ctx, id)
	if err != nil {
		return wrapCfg(err)
	}
	if !ok {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "unknown oauth provider: "+id)
	}
	if err := s.configService.SetOAuthProvider(ctx, id, in); err != nil {
		return wrapCfg(err)
	}
	return nil
}

// maskSecret 将非空密钥替换为掩码占位符，避免密钥回传前端。
func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	return sysconfig.SecretMask
}

func wrapCfg(err error) *apperrors.AppError {
	return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "config operation failed", err)
}
