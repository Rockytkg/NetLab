package radiusd

import (
	"context"
	"fmt"
	"os"

	"netlab-backend/internal/model"
)

// loadCertMaterial 解析服务端证书材料：certID 引用 nb_radius_certs 中的
// 托管证书（私钥密文在此解密）；为 0 时回退到环境变量配置的文件路径。
func (s *RadiusService) loadCertMaterial(certID uint64, certFile, keyFile string) (certPEM, keyPEM []byte, err error) {
	if certID > 0 {
		cert, err := s.loadDBCert(certID)
		if err != nil {
			return nil, nil, err
		}
		if cert.KeyPEM == "" {
			return nil, nil, fmt.Errorf("managed certificate %d (%s) has no private key", certID, cert.Name)
		}
		key, err := s.cipher.Decrypt(cert.KeyPEM)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt private key of managed certificate %d: %w", certID, err)
		}
		return []byte(cert.CertPEM), []byte(key), nil
	}

	if certFile == "" || keyFile == "" {
		return nil, nil, fmt.Errorf("no certificate configured (neither managed certificate nor cert/key file paths)")
	}
	certPEM, err = os.ReadFile(certFile) //nolint:gosec // 路径来自受信配置
	if err != nil {
		return nil, nil, fmt.Errorf("read cert file: %w", err)
	}
	keyPEM, err = os.ReadFile(keyFile) //nolint:gosec // 路径来自受信配置
	if err != nil {
		return nil, nil, fmt.Errorf("read key file: %w", err)
	}
	return certPEM, keyPEM, nil
}

// loadCAMaterial 解析 CA 证书材料：caID 引用托管 CA 证书；为 0 时回退到文件
// 路径；两者都未配置时返回 nil（调用方按"不校验对端 CA"处理）。
func (s *RadiusService) loadCAMaterial(caID uint64, caFile string) ([]byte, error) {
	if caID > 0 {
		cert, err := s.loadDBCert(caID)
		if err != nil {
			return nil, err
		}
		return []byte(cert.CertPEM), nil
	}
	if caFile == "" {
		return nil, nil
	}
	caPEM, err := os.ReadFile(caFile) //nolint:gosec // 路径来自受信配置
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	return caPEM, nil
}

// loadDBCert 按 ID 加载托管证书。
func (s *RadiusService) loadDBCert(id uint64) (*model.RadiusCert, error) {
	if s.CertRepo == nil {
		return nil, fmt.Errorf("certificate repository is not configured")
	}
	cert, err := s.CertRepo.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, fmt.Errorf("managed certificate %d not found", id)
	}
	return cert, nil
}
