package radius

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"

	dtorequest "netlab-backend/internal/dto/request"
	"netlab-backend/internal/model"
	"netlab-backend/pkg/apperrors"
)

// ListCerts 分页查询 TLS 证书（回填 HasKey）。
func (s *Service) ListCerts(ctx context.Context, page, size int, keyword, certType string) ([]model.RadiusCert, int64, *apperrors.AppError) {
	items, total, err := s.certs.List(ctx, page, size, keyword, certType)
	if err != nil {
		return nil, 0, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to list radius certs", err)
	}
	for i := range items {
		items[i].HasKey = items[i].KeyPEM != ""
	}
	return items, total, nil
}

// CreateCert 导入一张 TLS 证书：解析证书元数据落库，私钥加密存储。
func (s *Service) CreateCert(ctx context.Context, req *dtorequest.RadiusCertUpsertRequest) (*model.RadiusCert, *apperrors.AppError) {
	name := strings.TrimSpace(req.Name)
	if existing, err := s.certs.GetByName(ctx, name); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check cert name", err)
	} else if existing != nil {
		return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "certificate name already exists")
	}

	certType := req.CertType
	if certType == "" {
		certType = model.RadiusCertTypeServer
	}

	certPEM := strings.TrimSpace(req.CertPEM)
	if certPEM == "" {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "证书内容不能为空")
	}
	leaf, appErr := parseLeafCert(certPEM)
	if appErr != nil {
		return nil, appErr
	}

	keyPEM := strings.TrimSpace(req.KeyPEM)
	// 服务器证书必须持有私钥（EAP 隧道/RadSec 均要用于 TLS 握手）。
	if certType == model.RadiusCertTypeServer && keyPEM == "" {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "服务器证书必须提供私钥")
	}
	encryptedKey, appErr := s.validateAndEncryptKey(certPEM, keyPEM)
	if appErr != nil {
		return nil, appErr
	}

	cert := &model.RadiusCert{
		Name:     name,
		CertType: certType,
		CertPEM:  certPEM,
		KeyPEM:   encryptedKey,
		Remark:   req.Remark,
	}
	fillCertMeta(cert, leaf)
	if err := s.certs.Create(ctx, cert); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to create radius cert", err)
	}
	cert.HasKey = cert.KeyPEM != ""
	return cert, nil
}

// UpdateCert 更新证书：名称/备注直接更新；certPem 非空时重新解析元数据；
// keyPem 非空时与（新或旧）certPem 校验配对后加密替换。证书类型不允许修改。
func (s *Service) UpdateCert(ctx context.Context, id uint64, req *dtorequest.RadiusCertUpsertRequest) (*model.RadiusCert, *apperrors.AppError) {
	cert, err := s.certs.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius cert", err)
	}
	if cert == nil {
		return nil, apperrors.New(apperrors.ErrCodeUserNotFound, "certificate not found")
	}
	if req.CertType != "" && req.CertType != cert.CertType {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "证书类型不允许修改")
	}

	name := strings.TrimSpace(req.Name)
	if cert.Name != name {
		if existing, err := s.certs.GetByName(ctx, name); err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check cert name", err)
		} else if existing != nil {
			return nil, apperrors.New(apperrors.ErrCodeDuplicateEntry, "certificate name already exists")
		}
	}
	cert.Name = name
	cert.Remark = req.Remark

	if certPEM := strings.TrimSpace(req.CertPEM); certPEM != "" {
		leaf, appErr := parseLeafCert(certPEM)
		if appErr != nil {
			return nil, appErr
		}
		cert.CertPEM = certPEM
		fillCertMeta(cert, leaf)
	}
	if keyPEM := strings.TrimSpace(req.KeyPEM); keyPEM != "" {
		encryptedKey, appErr := s.validateAndEncryptKey(cert.CertPEM, keyPEM)
		if appErr != nil {
			return nil, appErr
		}
		cert.KeyPEM = encryptedKey
	} else if strings.TrimSpace(req.CertPEM) != "" && cert.KeyPEM != "" {
		// 仅替换了证书而未替换私钥：校验既有私钥仍与新证书配对，
		// 防止留下"证书新、私钥旧"的不可用组合。
		oldKey, decErr := s.cipher.Decrypt(cert.KeyPEM)
		if decErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to decrypt cert key", decErr)
		}
		if _, err := tls.X509KeyPair([]byte(cert.CertPEM), []byte(oldKey)); err != nil {
			return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "新证书与既有私钥不匹配，请一并替换私钥")
		}
	}

	if err := s.certs.Update(ctx, cert); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to update radius cert", err)
	}
	cert.HasKey = cert.KeyPEM != ""
	return cert, nil
}

// DeleteCert 删除证书；被 RADIUS 设置（EAP/RadSec 证书引用）引用时禁止删除。
func (s *Service) DeleteCert(ctx context.Context, id uint64) *apperrors.AppError {
	cert, err := s.certs.GetByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius cert", err)
	}
	if cert == nil {
		return apperrors.New(apperrors.ErrCodeUserNotFound, "certificate not found")
	}
	if appErr := s.checkCertReferenced(ctx, id); appErr != nil {
		return appErr
	}
	if err := s.certs.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to delete radius cert", err)
	}
	return nil
}

// ExportCert 导出证书 PEM；includeKey 且持有私钥时在证书后追加解密后的私钥 PEM。
func (s *Service) ExportCert(ctx context.Context, id uint64, includeKey bool) (string, []byte, *apperrors.AppError) {
	cert, err := s.certs.GetByID(ctx, id)
	if err != nil {
		return "", nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to load radius cert", err)
	}
	if cert == nil {
		return "", nil, apperrors.New(apperrors.ErrCodeUserNotFound, "certificate not found")
	}

	content := cert.CertPEM
	if includeKey && cert.KeyPEM != "" {
		keyPEM, err := s.cipher.Decrypt(cert.KeyPEM)
		if err != nil {
			return "", nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to decrypt cert key", err)
		}
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += keyPEM
	}
	return cert.Name + ".pem", []byte(content), nil
}

// checkCertReferenced 检查证书是否被 radius.eap / radius.system 配置 blob 引用。
func (s *Service) checkCertReferenced(ctx context.Context, id uint64) *apperrors.AppError {
	if eap, ok, err := s.cfgSvc.RadiusEap(ctx); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check cert references", err)
	} else if ok && (eap.TLSServerCertID == id || eap.TLSClientCAID == id) {
		return apperrors.New(apperrors.ErrCodeResourceInUse, "证书正被引用，请先在配置中取消引用")
	}
	if sys, ok, err := s.cfgSvc.RadiusSystem(ctx); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInternal, "failed to check cert references", err)
	} else if ok && (sys.RadsecCertID == id || sys.RadsecCACertID == id) {
		return apperrors.New(apperrors.ErrCodeResourceInUse, "证书正被引用，请先在配置中取消引用")
	}
	return nil
}

// parseLeafCert 从 PEM 文本中解析首个 CERTIFICATE 块（叶子证书）。
func parseLeafCert(certPEM string) (*x509.Certificate, *apperrors.AppError) {
	rest := []byte(certPEM)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "证书解析失败: "+err.Error())
		}
		return cert, nil
	}
	return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "无效的证书 PEM：未找到 CERTIFICATE 块")
}

// fillCertMeta 用解析出的叶子证书填充 Subject/Issuer/Serial/Fingerprint 与有效期。
// Serial 与 Fingerprint（SHA-256）均为大写 hex（无分隔符）。
func fillCertMeta(c *model.RadiusCert, leaf *x509.Certificate) {
	c.Subject = leaf.Subject.String()
	c.Issuer = leaf.Issuer.String()
	c.Serial = strings.ToUpper(hex.EncodeToString(leaf.SerialNumber.Bytes()))
	sum := sha256.Sum256(leaf.Raw)
	c.Fingerprint = strings.ToUpper(hex.EncodeToString(sum[:]))
	c.NotBefore = leaf.NotBefore
	c.NotAfter = leaf.NotAfter
}

// validateAndEncryptKey 校验私钥与证书配对并加密存储；keyPEM 为空时返回空串。
func (s *Service) validateAndEncryptKey(certPEM, keyPEM string) (string, *apperrors.AppError) {
	if keyPEM == "" {
		return "", nil
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		return "", apperrors.New(apperrors.ErrCodeInvalidRequest, "私钥与证书不匹配: "+err.Error())
	}
	encrypted, err := s.cipher.Encrypt(keyPEM)
	if err != nil {
		return "", apperrors.Wrap(apperrors.ErrCodeInternal, "failed to encrypt cert key", err)
	}
	return encrypted, nil
}
