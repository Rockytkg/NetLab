package radiusd

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"netlab-backend/internal/model"
	radiuserrors "netlab-backend/internal/radiusd/errors"
	"netlab-backend/internal/radiusd/plugins/accounting"
	"netlab-backend/internal/radiusd/registry"
	"netlab-backend/internal/radiusd/vendors"
)

// AcctService 处理 RADIUS Accounting-Request（记账请求）。
type AcctService struct {
	*RadiusService
}

// NewAcctService 构造记账服务。
func NewAcctService(radiusService *RadiusService) *AcctService {
	return &AcctService{RadiusService: radiusService}
}

// ServeRADIUS 实现 radius.Handler 接口，处理单个 Accounting-Request。
// 处理完成后同步应答 Accounting-Response。
func (s *AcctService) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	// 仅兜底未预期 panic；正常错误走错误返回路径。
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Error("radius accounting 处理 panic",
				zap.Any("recover", rec),
				zap.Stack("stacktrace"),
			)
		}
	}()

	if r == nil {
		return
	}

	// 匹配 NAS（源 IP 优先，NAS-Identifier 兜底）。
	nasrip := r.RemoteAddr.String()
	if host, _, err := net.SplitHostPort(nasrip); err == nil {
		nasrip = host
	}
	identifier := rfc2865.NASIdentifier_GetString(r.Packet)
	nas, err := s.GetNas(nasrip, identifier)
	if err != nil {
		s.logger.Warn("radius accounting 拒绝：未授权 NAS",
			zap.String("nasip", nasrip),
			zap.Error(err),
		)
		return
	}
	r.Secret = []byte(nas.Secret)

	statusType := rfc2866.AcctStatusType_Get(r.Packet)

	// 除 On/Off 外必须有用户名。
	var username string
	if statusType != rfc2866.AcctStatusType_Value_AccountingOn &&
		statusType != rfc2866.AcctStatusType_Value_AccountingOff {
		username = rfc2865.UserName_GetString(r.Packet)
		if username == "" {
			s.logger.Warn("radius accounting 拒绝：用户名为空",
				zap.String("nasip", nasrip),
				zap.Error(radiuserrors.NewAcctUsernameEmptyError()),
			)
			return
		}
	}

	// 校验 keyed Request Authenticator（RFC 2866）；不匹配说明报文伪造或
	// NAS 密钥配置错误，必须在变更任何会话状态前丢弃。
	if err := s.CheckRequestSecret(r.Packet, []byte(nas.Secret)); err != nil {
		s.logger.Warn("radius accounting 拒绝：签名校验失败",
			zap.String("nasip", nasrip),
			zap.String("username", username),
			zap.Error(err),
		)
		return
	}

	vendorReq := s.ParseVendor(r, nas.VendorCode)

	// 同步处理记账，完成后统一应答；整个处理链共享一个 DB 超时预算。
	ctx, cancel := context.WithTimeout(context.Background(), dbCallTimeout)
	defer cancel()
	if err := s.HandleAccounting(ctx, r, vendorReq, username, nas, nasrip); err != nil {
		s.logger.Error("radius accounting 处理失败",
			zap.String("username", username),
			zap.Int("status_type", int(statusType)),
			zap.Error(err),
		)
	}

	resp := r.Response(radius.CodeAccountingResponse)
	if err := w.Write(resp); err != nil {
		s.logger.Error("radius 写出 accounting-response 失败", zap.Error(err))
	}
}

// HandleAccounting 按 Acct-Status-Type 分发到注册的记账处理器。
func (s *AcctService) HandleAccounting(
	ctx context.Context,
	r *radius.Request,
	vendorReq *vendors.VendorRequest,
	username string,
	nas *model.RadiusNas,
	nasIP string,
) error {
	statusType := rfc2866.AcctStatusType_Get(r.Packet)

	acctCtx := &accounting.AccountingContext{
		Context:    ctx,
		Request:    r,
		VendorReq:  vendorReq,
		Username:   username,
		NAS:        nas,
		NASIP:      nasIP,
		StatusType: int(statusType),
	}

	handlers := registry.GetAccountingHandlers()
	for _, handler := range handlers {
		if handler.CanHandle(acctCtx) {
			return handler.Handle(acctCtx)
		}
	}
	return fmt.Errorf("no handler found for status type %d", statusType)
}
