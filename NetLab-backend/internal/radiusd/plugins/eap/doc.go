// Package eap provides the shared EAP authentication orchestration layer for
// ToughRADIUS.
//
// It defines message/state abstractions, method handler interfaces, and the
// coordinator that dispatches EAP rounds to registered handlers while
// preserving per-session state across RADIUS request/response exchanges.
//
// 本包移植自 github.com/talkincode/toughradius（MIT License）。
package eap
