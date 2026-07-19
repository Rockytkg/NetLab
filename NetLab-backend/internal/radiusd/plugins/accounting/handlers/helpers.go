package handlers

import (
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

// na is the empty-value placeholder used for session/accounting string fields
// whose value cannot be obtained (mirrors radiusd.NA; kept local to avoid an
// import cycle).
const na = "N/A"

// ifEmptyStr returns def when s is empty, otherwise s (mirrors
// radiusd.IfEmptyStr; kept local to avoid an import cycle).
func ifEmptyStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// classAttrOrNA 读取 Class 属性（RFC 2865 #25，NAS 在记账报文中回传的不透明值），
// 缺失时返回 na。
func classAttrOrNA(p *radius.Packet) string {
	if v := rfc2865.Class_Get(p); len(v) > 0 {
		return string(v)
	}
	return na
}

// terminateCauseString 返回 Acct-Terminate-Cause（RFC 2866 #49）的字符串名，
// 缺失或未知时返回空串。
func terminateCauseString(p *radius.Packet) string {
	cause := rfc2866.AcctTerminateCause_Get(p)
	if cause == 0 {
		return ""
	}
	return cause.String()
}
