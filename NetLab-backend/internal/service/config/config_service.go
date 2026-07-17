// Package sysconfig 提供运行时系统配置服务。
//
// 它是 OAuth、SMTP 与安全策略等配置的单一事实来源：所有配置以
// key-value 形式持久化在 system_configs 表中，敏感字段（SMTP 密码、
// OAuth Client Secret）经 AES-256-GCM 加密存储。服务在内存中缓存
// 配置快照，并在每次写入后失效缓存，从而让管理端的修改无需重启即可
// 热生效。
package sysconfig

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"

	"netlab-backend/internal/repository"
	"netlab-backend/pkg/crypto"
)

// ─── 配置键 ──────────────────────────────────────────────────────────

const (
	keyRegistrationEnabled  = "registration_enabled"
	keyCaptchaEnabled       = "captcha_enabled"
	keyPasskeyEnabled       = "passkey_enabled"
	keyPasswordResetEnabled = "password_reset_enabled"
	keyTwoFactorRequired    = "two_factor_required"
	keyPasswordMaxAgeDays   = "password_max_age_days"
	keyICPBeian             = "icp_beian"
	keyPoliceBeian          = "police_beian"
	keySMTP                 = "smtp"
	keyOAuthPrefix          = "oauth." // oauth.<providerID>
)

// SecretMask 是回传给前端以代替已配置密钥的占位符。
// 前端提交更新时若原样带回该占位符（或留空），后端将保留既有密钥。
const SecretMask = "__UNCHANGED__"

// ─── 配置分组结构 ────────────────────────────────────────────────────

// SecuritySettings 保存登录/注册相关的安全策略开关。
type SecuritySettings struct {
	RegistrationEnabled  bool `json:"registrationEnabled"`
	CaptchaEnabled       bool `json:"captchaEnabled"`
	PasskeyEnabled       bool `json:"passkeyEnabled"`
	PasswordResetEnabled bool `json:"passwordResetEnabled"`
	TwoFactorRequired    bool `json:"twoFactorRequired"`
	PasswordMaxAgeDays   int  `json:"passwordMaxAgeDays"`
}

// BeianSettings 保存备案信息。
type BeianSettings struct {
	ICPBeian    string `json:"icpBeian"`
	PoliceBeian string `json:"policeBeian"`
}

// SMTPSettings 保存邮件服务配置。Password 在内部为明文，
// 持久化时加密；对外展示时被掩码。
type SMTPSettings struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"useTls"`
}

// IsConfigured 在 SMTP 已启用且必填字段齐全时返回 true。
func (s SMTPSettings) IsConfigured() bool {
	return s.Enabled && s.Host != "" && s.Port > 0 && s.From != ""
}

// ProviderSettings 保存单个 OAuth 提供商的可编辑配置。
type ProviderSettings struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURL  string `json:"redirectUrl"`
}

// IsConfigured 在提供商已启用且填写了 client 凭据时返回 true。
func (p ProviderSettings) IsConfigured() bool {
	return p.Enabled && p.ClientID != "" && p.ClientSecret != ""
}

// ─── 服务 ────────────────────────────────────────────────────────────

// Service 是运行时系统配置服务。
type Service struct {
	repo   *repository.ConfigRepository
	cipher *crypto.AESCipher
	redis  *redis.Client

	mu    sync.RWMutex
	cache map[string]string // 原始 key→value 快照；nil 表示未加载
}

// NewService 创建一个新的配置服务。
func NewService(repo *repository.ConfigRepository, cipher *crypto.AESCipher, clients ...*redis.Client) *Service {
	var client *redis.Client
	if len(clients) > 0 {
		client = clients[0]
	}
	s := &Service{repo: repo, cipher: cipher, redis: client}
	if client != nil {
		go s.listenInvalidation()
	}
	return s
}

// ─── 缓存管理 ────────────────────────────────────────────────────────

// snapshot 返回配置快照，必要时从数据库加载并填充缓存。
func (s *Service) snapshot(ctx context.Context) (map[string]string, error) {
	s.mu.RLock()
	if s.cache != nil {
		cached := s.cache
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	values, err := s.repo.GetAllValues(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache = values
	s.mu.Unlock()
	return values, nil
}

// invalidate 清空缓存，使下次读取重新从数据库加载。
func (s *Service) invalidate() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}

// set 持久化一个 key 并使缓存失效。
func (s *Service) set(ctx context.Context, key, value, description string) error {
	if err := s.repo.SetValue(ctx, key, value, description); err != nil {
		return err
	}
	s.invalidate()
	if s.redis != nil {
		_ = s.redis.Publish(ctx, "netlab:config:invalidate", key).Err()
	}
	return nil
}

func (s *Service) listenInvalidation() {
	sub := s.redis.Subscribe(context.Background(), "netlab:config:invalidate")
	defer sub.Close()
	if _, err := sub.Receive(context.Background()); err != nil {
		return
	}
	for range sub.Channel() {
		s.invalidate()
	}
}

// ─── 安全策略 ────────────────────────────────────────────────────────

// Security 返回当前安全策略；未配置的项采用安全默认值。
func (s *Service) Security(ctx context.Context) (SecuritySettings, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return SecuritySettings{}, err
	}
	return SecuritySettings{
		RegistrationEnabled:  getBool(snap, keyRegistrationEnabled, true),
		CaptchaEnabled:       getBool(snap, keyCaptchaEnabled, false),
		PasskeyEnabled:       getBool(snap, keyPasskeyEnabled, true),
		PasswordResetEnabled: getBool(snap, keyPasswordResetEnabled, true),
		TwoFactorRequired:    getBool(snap, keyTwoFactorRequired, false),
		PasswordMaxAgeDays:   getInt(snap, keyPasswordMaxAgeDays, 0),
	}, nil
}

// SetSecurity 更新安全策略并热生效。
func (s *Service) SetSecurity(ctx context.Context, in SecuritySettings) error {
	updates := []struct{ key, val, desc string }{
		{keyRegistrationEnabled, boolStr(in.RegistrationEnabled), "Allow new user registration"},
		{keyCaptchaEnabled, boolStr(in.CaptchaEnabled), "Require image captcha on login and registration"},
		{keyPasskeyEnabled, boolStr(in.PasskeyEnabled), "Enable WebAuthn/Passkey authentication"},
		{keyPasswordResetEnabled, boolStr(in.PasswordResetEnabled), "Enable password reset via email"},
		{keyTwoFactorRequired, boolStr(in.TwoFactorRequired), "Require two-factor authentication for backend access"},
		{keyPasswordMaxAgeDays, intStr(nonNegative(in.PasswordMaxAgeDays)), "Password max age in days; 0 means never expires"},
	}
	for _, u := range updates {
		if err := s.repo.SetValue(ctx, u.key, u.val, u.desc); err != nil {
			s.invalidate()
			return err
		}
	}
	s.invalidate()
	if s.redis != nil {
		_ = s.redis.Publish(ctx, "netlab:config:invalidate", "security").Err()
	}
	return nil
}

// ─── 备案信息 ────────────────────────────────────────────────────────

// EncryptSecret 使用配置主密钥对敏感值进行 AES-GCM 加密。
func (s *Service) EncryptSecret(plaintext string) (string, error) {
	return s.cipher.Encrypt(plaintext)
}

// DecryptSecret 解密由 EncryptSecret 生成的密文。
func (s *Service) DecryptSecret(stored string) (string, error) {
	return s.cipher.Decrypt(stored)
}

// Beian 返回备案信息。
func (s *Service) Beian(ctx context.Context) (BeianSettings, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return BeianSettings{}, err
	}
	return BeianSettings{
		ICPBeian:    getString(snap, keyICPBeian),
		PoliceBeian: getString(snap, keyPoliceBeian),
	}, nil
}

// SetBeian 更新备案信息。
func (s *Service) SetBeian(ctx context.Context, in BeianSettings) error {
	if err := s.set(ctx, keyICPBeian, jsonString(in.ICPBeian), "ICP filing number shown on the login page"); err != nil {
		return err
	}
	return s.set(ctx, keyPoliceBeian, jsonString(in.PoliceBeian), "Public-security (公安) filing number shown on the login page")
}

// ─── SMTP ────────────────────────────────────────────────────────────

// SMTP 返回解密后的 SMTP 配置（供邮件服务使用）。
func (s *Service) SMTP(ctx context.Context) (SMTPSettings, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return SMTPSettings{}, err
	}

	var out SMTPSettings
	if raw, ok := snap[keySMTP]; ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if out.Port == 0 {
		out.Port = 587
	}

	// 解密密码字段
	if out.Password != "" {
		plain, err := s.cipher.Decrypt(out.Password)
		if err != nil {
			return SMTPSettings{}, err
		}
		out.Password = plain
	}
	return out, nil
}

// SetSMTP 更新 SMTP 配置。若 in.Password 为空或为掩码占位符，
// 则保留既有加密密码（避免前端因掩码而误清空密钥）。
func (s *Service) SetSMTP(ctx context.Context, in SMTPSettings) error {
	// 解析既有密文以便在密码留空时保留。
	snap, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	var existing SMTPSettings
	if raw, ok := snap[keySMTP]; ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &existing)
	}

	stored := in
	if in.Password == "" || in.Password == SecretMask {
		stored.Password = existing.Password // 已是密文，原样保留
	} else {
		enc, err := s.cipher.Encrypt(in.Password)
		if err != nil {
			return err
		}
		stored.Password = enc
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return s.set(ctx, keySMTP, string(data), "SMTP email server settings")
}

// ─── OAuth ───────────────────────────────────────────────────────────

// OAuthProvider 返回指定提供商解密后的配置。
// 第二个返回值指示该提供商是否存在于注册表中。
func (s *Service) OAuthProvider(ctx context.Context, id string) (ProviderSettings, bool, error) {
	if _, ok := providerMeta(id); !ok {
		return ProviderSettings{}, false, nil
	}

	snap, err := s.snapshot(ctx)
	if err != nil {
		return ProviderSettings{}, false, err
	}

	out := s.parseProvider(snap, id)
	if out.ClientSecret != "" {
		plain, err := s.cipher.Decrypt(out.ClientSecret)
		if err != nil {
			return ProviderSettings{}, false, err
		}
		out.ClientSecret = plain
	}
	return out, true, nil
}

// parseProvider 从快照中读取某提供商的原始（可能加密）配置。
func (s *Service) parseProvider(snap map[string]string, id string) ProviderSettings {
	var out ProviderSettings
	if raw, ok := snap[keyOAuthPrefix+id]; ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

// SetOAuthProvider 更新某 OAuth 提供商配置。若 ClientSecret 留空或为
// 掩码占位符，则保留既有密文。
func (s *Service) SetOAuthProvider(ctx context.Context, id string, in ProviderSettings) error {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	existing := s.parseProvider(snap, id)

	stored := in
	if in.ClientSecret == "" || in.ClientSecret == SecretMask {
		stored.ClientSecret = existing.ClientSecret
	} else {
		enc, err := s.cipher.Encrypt(in.ClientSecret)
		if err != nil {
			return err
		}
		stored.ClientSecret = enc
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return s.set(ctx, keyOAuthPrefix+id, string(data), "OAuth provider: "+id)
}

// EnabledProviders 返回已启用且配置完整的 OAuth 提供商及其登录 URL，
// 供登录页展示。
func (s *Service) EnabledProviders(ctx context.Context) ([]repository.OAuthProvider, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}

	var result []repository.OAuthProvider
	for _, meta := range providerRegistry {
		cfg := s.parseProvider(snap, meta.ID)
		if !cfg.Enabled || cfg.ClientID == "" {
			continue
		}
		authURL := buildAuthURL(meta, cfg)
		if authURL == "" {
			continue
		}
		result = append(result, repository.OAuthProvider{
			ID:      meta.ID,
			Name:    meta.Name,
			Icon:    meta.Icon,
			Color:   meta.Color,
			AuthURL: authURL,
		})
	}
	return result, nil
}

// ─── 登录页公开配置 ──────────────────────────────────────────────────

// PublicConfig 编译登录页所需的公开系统配置。
func (s *Service) PublicConfig(ctx context.Context) (*repository.SystemConfigResult, error) {
	sec, err := s.Security(ctx)
	if err != nil {
		return nil, err
	}
	beian, err := s.Beian(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.EnabledProviders(ctx)
	if err != nil {
		return nil, err
	}

	return &repository.SystemConfigResult{
		RegistrationEnabled:  sec.RegistrationEnabled,
		CaptchaEnabled:       sec.CaptchaEnabled,
		PasskeyEnabled:       sec.PasskeyEnabled,
		PasswordResetEnabled: sec.PasswordResetEnabled,
		TwoFactorRequired:    sec.TwoFactorRequired,
		OAuthProviders:       providers,
		ICPBeian:             beian.ICPBeian,
		PoliceBeian:          beian.PoliceBeian,
	}, nil
}

// ─── 解析辅助 ────────────────────────────────────────────────────────

func getBool(snap map[string]string, key string, def bool) bool {
	v, ok := snap[key]
	if !ok || v == "" {
		return def
	}
	switch v {
	case "true", `"true"`, "1":
		return true
	case "false", `"false"`, "0":
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return def
}

func getString(snap map[string]string, key string) string {
	v, ok := snap[key]
	if !ok || v == "" {
		return ""
	}
	var out string
	if err := json.Unmarshal([]byte(v), &out); err == nil {
		return out
	}
	return v
}

func getInt(snap map[string]string, key string, def int) int {
	v, ok := snap[key]
	if !ok || v == "" {
		return def
	}
	var asJSON int
	if err := json.Unmarshal([]byte(v), &asJSON); err == nil {
		return asJSON
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return def
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func intStr(n int) string {
	return strconv.Itoa(n)
}

func nonNegative(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// jsonString 将字符串编码为 jsonb 可存储的带引号 JSON 字符串。
func jsonString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}
