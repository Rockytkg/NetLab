package parsers

import (
	"strings"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"netlab-backend/internal/radiusd/plugins/vendorparsers"
)

// DefaultParser is the default vendor attribute parser
type DefaultParser struct{}

func (p *DefaultParser) VendorCode() string {
	return "default"
}

func (p *DefaultParser) VendorName() string {
	return "Standard"
}

func (p *DefaultParser) Parse(r *radius.Request) (*vendorparsers.VendorRequest, error) {
	vr := &vendorparsers.VendorRequest{}

	// ParseMACaddresses
	macval := rfc2865.CallingStationID_GetString(r.Packet)
	if macval != "" {
		vr.MacAddr = strings.ReplaceAll(macval, "-", ":")
	}

	// The default parser does not parse VLAN attributes
	vr.Vlanid1 = 0
	vr.Vlanid2 = 0

	return vr, nil
}
