// Package vendorparsers provides shared helpers for vendor attribute parsers.
//
// 本包移植自 github.com/talkincode/toughradius（MIT License）。
//
// The VendorRequest/VendorParser/VendorResponseBuilder types live in the
// internal/radiusd/vendors package; they are re-exported here as aliases so
// the ported parsers keep their original references.
package vendorparsers

import (
	"netlab-backend/internal/radiusd/vendors"
)

// VendorRequest holds vendor-specific request data.
type VendorRequest = vendors.VendorRequest

// VendorParser defines the vendor attribute parser interface.
type VendorParser = vendors.VendorParser

// VendorResponseBuilder defines vendor response builders.
type VendorResponseBuilder = vendors.VendorResponseBuilder
