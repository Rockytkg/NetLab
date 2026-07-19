package radiusd

import (
	"go.uber.org/zap"
	"layeh.com/radius"

	"netlab-backend/internal/radiusd/vendors"
)

// ParseVendor 按 NAS 厂商代码解析请求中的厂商属性（MAC/VLAN）。
// 未匹配的厂商回落到 default 解析器。
func (s *RadiusService) ParseVendor(r *radius.Request, vendorCode string) *vendors.VendorRequest {
	parser, ok := vendors.GetParser(vendorCode)
	if !ok {
		parser, ok = vendors.GetParser("default")
		if !ok {
			s.logger.Error("radius 默认厂商解析器未注册")
			return &vendors.VendorRequest{}
		}
	}

	vendorReq, err := parser.Parse(r)
	if err != nil {
		s.logger.Error("radius 厂商属性解析失败",
			zap.String("vendor_code", vendorCode),
			zap.Error(err),
		)
		return &vendors.VendorRequest{}
	}
	return &vendors.VendorRequest{
		MacAddr: vendorReq.MacAddr,
		Vlanid1: vendorReq.Vlanid1,
		Vlanid2: vendorReq.Vlanid2,
	}
}
