package radiusd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"layeh.com/radius"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/plugins/eap"
	"netlab-backend/internal/radiusd/plugins/eap/handlers"
	"netlab-backend/internal/radiusd/plugins/eap/statemanager"
	"netlab-backend/internal/radiusd/registry"
	"netlab-backend/internal/radiusd/repository"
	"netlab-backend/internal/radiusd/vendors"
	"netlab-backend/pkg/crypto"
)

// EAPAuthHelper 是 EAP 认证协调入口（多轮挑战由 State 属性驱动）。
type EAPAuthHelper struct {
	coordinator *eap.Coordinator
}

// NewEAPAuthHelper 构造 EAP 认证辅助器。
// 密码来源为本地用户（加载时已解密为明文）；TLS 隧道证书来自配置文件路径。
func NewEAPAuthHelper(radiusService *RadiusService) *EAPAuthHelper {
	stateManager := statemanager.NewMemoryStateManager()
	coordinator := eap.NewCoordinator(stateManager, localPasswordProvider{}, registry.GetGlobalRegistry(), false)
	return &EAPAuthHelper{coordinator: coordinator}
}

// HandleEAPAuthentication 处理 EAP 认证，返回 (handled, success, err)。
func (h *EAPAuthHelper) HandleEAPAuthentication(
	w radius.ResponseWriter,
	r *radius.Request,
	user *model.RadiusUser,
	nas *model.RadiusNas,
	vendorReq *vendors.VendorRequest,
	response *radius.Packet,
	eapMethod string,
) (handled bool, success bool, err error) {
	isMacAuth := vendorReq.MacAddr != "" && vendorReq.MacAddr == user.Username
	return h.coordinator.HandleEAPRequest(w, r, user, nas, response, nas.Secret, isMacAuth, eapMethod)
}

// SendEAPSuccess 发送 EAP-Success。
func (h *EAPAuthHelper) SendEAPSuccess(w radius.ResponseWriter, r *radius.Request, response *radius.Packet, secret string) error {
	return h.coordinator.SendEAPSuccess(w, r, response, secret)
}

// SendEAPFailure 发送 EAP-Failure。
func (h *EAPAuthHelper) SendEAPFailure(w radius.ResponseWriter, r *radius.Request, secret string, reason error) error {
	return h.coordinator.SendEAPFailure(w, r, secret, reason)
}

// CleanupState 清理请求的 EAP 挑战状态。
func (h *EAPAuthHelper) CleanupState(r *radius.Request) {
	h.coordinator.CleanupState(r)
}

// localPasswordProvider 提供本地用户的明文密码（用户加载时已解密）。
type localPasswordProvider struct{}

// GetPassword 返回用户的明文密码；MAC 认证时以 MAC 地址作为密码。
func (localPasswordProvider) GetPassword(user *model.RadiusUser, isMacAuth bool) (string, error) {
	if isMacAuth {
		if user.MacAddr != "" {
			return user.MacAddr, nil
		}
		return user.Username, nil
	}
	return user.Password, nil
}

// —— EAP TLS 隧道证书的解析实现（DB 托管证书优先，环境变量文件路径回退）——

// 文件证书解析器识别的固定名称（对应环境变量中的文件路径配置）。
const (
	fileServerCertName = "file:eap-server"
	fileClientCAName   = "file:eap-client-ca"
)

// certMaterialResolver 按设置读取器给出的证书名解析 PEM 材料：
// 十进制数字表示 nb_radius_certs 中的证书 ID（私钥密文运行时解密）；
// "file:" 前缀表示环境变量配置的文件路径（向后兼容）。
// 每次 EAP TLS 握手都会重新解析，因此管理端替换证书或修改引用后立即生效。
type certMaterialResolver struct {
	cfgFn  func() config.RadiusConfig
	repo   repository.CertRepository
	cipher *crypto.AESCipher
}

// ServerKeyPair 返回服务端 PEM 证书链与私钥。
func (r *certMaterialResolver) ServerKeyPair(name string) (certPEM, keyPEM []byte, err error) {
	cfg := r.cfgFn()
	switch {
	case name == fileServerCertName:
		certPEM, err = os.ReadFile(cfg.EAPCertFile) //nolint:gosec // 路径来自受信配置
		if err != nil {
			return nil, nil, fmt.Errorf("read EAP server cert: %w", err)
		}
		keyPEM, err = os.ReadFile(cfg.EAPKeyFile) //nolint:gosec // 路径来自受信配置
		if err != nil {
			return nil, nil, fmt.Errorf("read EAP server key: %w", err)
		}
		return certPEM, keyPEM, nil
	default:
		cert, err := r.loadDBCert(name)
		if err != nil {
			return nil, nil, fmt.Errorf("load EAP server cert: %w", err)
		}
		if cert.KeyPEM == "" {
			return nil, nil, fmt.Errorf("server certificate %q has no private key", name)
		}
		key, err := r.cipher.Decrypt(cert.KeyPEM)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt private key of certificate %q: %w", name, err)
		}
		return []byte(cert.CertPEM), []byte(key), nil
	}
}

// CABundle 返回客户端 CA PEM 包。
func (r *certMaterialResolver) CABundle(name string) (caPEM []byte, err error) {
	cfg := r.cfgFn()
	if name == fileClientCAName {
		caPEM, err = os.ReadFile(cfg.EAPCAFile) //nolint:gosec // 路径来自受信配置
		if err != nil {
			return nil, fmt.Errorf("read EAP client CA: %w", err)
		}
		return caPEM, nil
	}
	cert, err := r.loadDBCert(name)
	if err != nil {
		return nil, fmt.Errorf("load EAP client CA: %w", err)
	}
	return []byte(cert.CertPEM), nil
}

// loadDBCert 按数字 ID 加载 DB 证书。
func (r *certMaterialResolver) loadDBCert(name string) (*model.RadiusCert, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("certificate repository is not configured")
	}
	id, err := strconv.ParseUint(name, 10, 64)
	if err != nil || id == 0 {
		return nil, fmt.Errorf("unknown certificate %q", name)
	}
	cert, err := r.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, fmt.Errorf("certificate %q not found", name)
	}
	return cert, nil
}

// eapTLSSettings 依据当前运行时配置为 handlers.TLSSettingsReader 提供
// 证书引用与 TLS 版本设置；每次握手重新读取，配置修改立即生效。
type eapTLSSettings struct {
	cfgFn func() config.RadiusConfig
}

// GetString 实现 handlers.TLSSettingsReader。
func (s *eapTLSSettings) GetString(category, name string) string {
	if category != "radius" {
		return ""
	}
	cfg := s.cfgFn()
	switch name {
	case handlers.SettingEapTlsServerCert:
		if cfg.EAPServerCertID > 0 {
			return strconv.FormatUint(cfg.EAPServerCertID, 10)
		}
		if cfg.EAPCertFile != "" && cfg.EAPKeyFile != "" {
			return fileServerCertName
		}
	case handlers.SettingEapTlsClientCa:
		if cfg.EAPClientCACertID > 0 {
			return strconv.FormatUint(cfg.EAPClientCACertID, 10)
		}
		if cfg.EAPCAFile != "" {
			return fileClientCAName
		}
	case handlers.SettingEapTlsMinVersion:
		if cfg.EAPTLSMinVersion != "" {
			return cfg.EAPTLSMinVersion
		}
		return "1.2"
	}
	return ""
}

// newEAPTLSProviders 构建 EAP-TLS/PEAP/TTLS 的 TLS 配置提供者（读取实时配置）。
func newEAPTLSProviders(s *RadiusService) (reader *eapTLSSettings, resolver *certMaterialResolver) {
	reader = &eapTLSSettings{cfgFn: s.cfg}
	resolver = &certMaterialResolver{cfgFn: s.cfg, repo: s.CertRepo, cipher: s.cipher}
	return reader, resolver
}

// eapMethodAllowed 判定配置的启用列表是否允许指定 EAP 方法（"*" 为全部）。
func eapMethodAllowed(enabledHandlers, method string) bool {
	enabledHandlers = strings.TrimSpace(enabledHandlers)
	if enabledHandlers == "" || enabledHandlers == "*" {
		return true
	}
	for _, h := range strings.Split(enabledHandlers, ",") {
		if strings.EqualFold(strings.TrimSpace(h), method) {
			return true
		}
	}
	return false
}

// resolveEAPMethod 归一化配置的优先 EAP 方法。
func resolveEAPMethod(preferred string) string {
	method := strings.TrimSpace(strings.ToLower(preferred))
	if method == "" {
		return "eap-md5"
	}
	return method
}
