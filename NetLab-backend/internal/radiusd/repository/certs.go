package repository

import (
	"context"

	"netlab-backend/internal/model"
)

// CertRepository 是 TLS 证书材料存储（nb_radius_certs）。
// 返回的证书 KeyPEM 仍为密文，由运行时持有加密组件的一方解密。
type CertRepository interface {
	// GetByID 按 ID 查询证书，不存在返回 (nil, nil)。
	GetByID(ctx context.Context, id uint64) (*model.RadiusCert, error)
}
