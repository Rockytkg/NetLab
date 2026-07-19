// Package accounting defines pluggable accounting handlers and shared request context types.
//
// 本包移植自 github.com/talkincode/toughradius（MIT License）。
package accounting

import (
	"context"

	"layeh.com/radius"

	"netlab-backend/internal/model"
	vendorparserspkg "netlab-backend/internal/radiusd/plugins/vendorparsers"
)

// AccountingContext is the shared accounting context
type AccountingContext struct {
	Context    context.Context
	Request    *radius.Request
	VendorReq  *vendorparserspkg.VendorRequest
	Username   string
	NAS        *model.RadiusNas
	NASIP      string
	StatusType int // rfc2866: Start=1, Stop=2, InterimUpdate=3, AccountingOn=7, AccountingOff=8
}

// AccountingHandler defines the accounting handler interface
type AccountingHandler interface {
	// Name Returnshandlernames
	Name() string

	// CanHandle determines whether the handler can process this accounting request
	CanHandle(ctx *AccountingContext) bool

	// Handle HandleAccountingrequest
	Handle(ctx *AccountingContext) error
}
