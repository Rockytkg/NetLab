package enhancers

import (
	"context"
	"math"
	"net"
	"strings"
	"time"

	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/rfc3162"
	"layeh.com/radius/rfc4818"
	"layeh.com/radius/rfc6911"

	"netlab-backend/internal/radiusd/plugins/auth"
)

// DefaultAcceptEnhancer sets standard RADIUS attributes
type DefaultAcceptEnhancer struct{}

func NewDefaultAcceptEnhancer() *DefaultAcceptEnhancer {
	return &DefaultAcceptEnhancer{}
}

func (e *DefaultAcceptEnhancer) Name() string {
	return "default-accept"
}

func (e *DefaultAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}

	user := authCtx.User
	response := authCtx.Response

	timeout := int64(time.Until(user.ExpireTime).Seconds())
	if timeout > math.MaxInt32 {
		timeout = math.MaxInt32
	}

	// The interim interval comes from request-scoped metadata (seconds);
	// fall back to 120 when the pipeline did not supply one.
	interim := int64(120)
	var timeoutCap int64
	if authCtx.Metadata != nil {
		if v, ok := authCtx.Metadata["acct_interim_interval"].(int); ok && v > 0 {
			interim = int64(v)
		}
		if v, ok := authCtx.Metadata["session_timeout_cap"].(int); ok && v > 0 {
			timeoutCap = int64(v)
		}
	}

	// Session-Timeout 取「用户过期倒计时」，并按管理端配置的上限收敛
	//（toughradius 的 radius.SessionTimeout 语义：0 表示不限制）。
	if timeoutCap > 0 && timeout > timeoutCap {
		timeout = timeoutCap
	}

	if timeout > 0 {
		_ = rfc2865.SessionTimeout_Set(response, rfc2865.SessionTimeout(timeout)) //nolint:errcheck,gosec // G115: timeout is validated
	}
	_ = rfc2869.AcctInterimInterval_Set(response, rfc2869.AcctInterimInterval(interim)) //nolint:errcheck,gosec // G115: interim is validated

	// Use getter method for AddrPool
	addrPool := user.GetAddrPool()
	if isNotEmptyAndNA(addrPool) {
		_ = rfc2869.FramedPool_SetString(response, addrPool) //nolint:errcheck
	}

	// User-specific IP address (always use direct access)
	if isNotEmptyAndNA(user.IpAddr) {
		_ = rfc2865.FramedIPAddress_Set(response, net.ParseIP(user.IpAddr)) //nolint:errcheck
	}

	// Set FramedIPv6Prefix if user has a fixed IPv6 address
	if isNotEmptyAndNA(user.IpV6Addr) {
		// IPv6 prefix format: address/prefix-length (e.g., "2001:db8::1/64")
		// If only address is provided, append /128 for single host
		ipv6Prefix := user.IpV6Addr
		if !strings.Contains(ipv6Prefix, "/") {
			ipv6Prefix = ipv6Prefix + "/128"
		}
		if _, ipnet, err := net.ParseCIDR(ipv6Prefix); err == nil {
			_ = rfc3162.FramedIPv6Prefix_Set(response, ipnet) //nolint:errcheck
		}

		// Set Framed-IPv6-Address (RFC 6911, Section 2.1) when the user has a
		// single static host address (bare address or an explicit /128). RFC 6911
		// defines this attribute to assign a complete IPv6 address, which is more
		// natural for DHCPv6 than the RFC 3162 Framed-Interface-Id +
		// Framed-IPv6-Prefix split, and permits it to coexist with
		// Framed-IPv6-Prefix in the same Access-Accept.
		if ip, ok := singleIPv6Host(user.IpV6Addr); ok {
			_ = rfc6911.FramedIPv6Address_Set(response, ip) //nolint:errcheck
		}
	}

	// Use getter method for IPv6PrefixPool
	ipv6Pool := user.GetIPv6PrefixPool()
	if isNotEmptyAndNA(ipv6Pool) {
		_ = rfc3162.FramedIPv6Pool_SetString(response, ipv6Pool) //nolint:errcheck
	}

	// Set Delegated-IPv6-Prefix (RFC 4818, attribute 123) when the user has a
	// statically delegated prefix for DHCPv6-PD. The value is a CIDR prefix such
	// as "2001:db8:1234::/48"; a bare address without a prefix length is treated
	// as a single-host /128 delegation. IPv4 and unparseable values are skipped
	// so a misconfiguration cannot break the Access-Accept or emit a malformed
	// (4-byte) prefix.
	if isNotEmptyAndNA(user.DelegatedIpv6Prefix) {
		delegated := user.DelegatedIpv6Prefix
		if !strings.Contains(delegated, "/") {
			delegated += "/128"
		}
		if _, ipnet, err := net.ParseCIDR(delegated); err == nil && ipnet.IP.To4() == nil {
			_ = rfc4818.DelegatedIPv6Prefix_Set(response, ipnet) //nolint:errcheck
		}
	}

	// Set Delegated-IPv6-Prefix-Pool (RFC 6911, attribute 171) so the NAS can
	// allocate a delegated prefix from a named DHCPv6-PD pool. Per RFC 6911 §2.4
	// this is distinct from the Framed-IPv6-Pool (SLAAC) set above; it uses the
	// dedicated getter that honors profile inheritance.
	delegatedPool := user.GetDelegatedIpv6PrefixPool()
	if isNotEmptyAndNA(delegatedPool) {
		_ = rfc6911.DelegatedIPv6PrefixPool_SetString(response, delegatedPool) //nolint:errcheck
	}

	return nil
}

// singleIPv6Host reports whether value designates a single IPv6 host address and
// returns the parsed address. It accepts either a bare IPv6 address
// ("2001:db8::1") or an explicit /128 prefix ("2001:db8::1/128"). It returns
// ok=false for IPv4 values, multi-host prefixes (for example /64, which are
// advertised as Framed-IPv6-Prefix instead), and unparseable input.
func singleIPv6Host(value string) (net.IP, bool) {
	addr := value
	if idx := strings.IndexByte(value, '/'); idx >= 0 {
		if value[idx+1:] != "128" {
			return nil, false
		}
		addr = value[:idx]
	}
	ip := net.ParseIP(addr)
	if ip == nil || ip.To4() != nil {
		return nil, false
	}
	return ip, true
}
