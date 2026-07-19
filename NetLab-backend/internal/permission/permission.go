// Package permission contains the application's authoritative permission registry.
//
// The registry is intentionally small and describes permissions that are actually
// used by protected HTTP routes. It is consumed both by the router and by RBAC
// synchronization, so the database cannot drift into a catalog of planned or
// nonexistent features.
package permission

// Definition describes one permission exposed by the application.
type Definition struct {
	Resource    string
	Action      string
	Description string
}

const (
	AuthRead      = "auth.read"
	RBACRead      = "rbac.read"
	RBACWrite     = "rbac.write"
	SettingRead   = "setting.read"
	SettingUpdate = "setting.update"
	UserRead      = "user.read"
	UserCreate    = "user.create"
	UserUpdate    = "user.update"
	UserDelete    = "user.delete"
	UserImport    = "user.import"
	LogRead       = "log.read"
	LogDelete     = "log.delete"
	RadiusRead    = "radius.read"
	RadiusManage  = "radius.manage"
)

// Catalog is the single source of truth for permissions that can be assigned.
// Keep this list aligned with the permission middleware calls in router.go.
var Catalog = []Definition{
	{Resource: "auth", Action: "read", Description: "Read the current account"},
	{Resource: "rbac", Action: "read", Description: "View RBAC configuration"},
	{Resource: "rbac", Action: "write", Description: "Modify RBAC configuration"},
	{Resource: "setting", Action: "read", Description: "View system settings"},
	{Resource: "setting", Action: "update", Description: "Update system settings"},
	{Resource: "user", Action: "read", Description: "View users"},
	{Resource: "user", Action: "create", Description: "Create users"},
	{Resource: "user", Action: "update", Description: "Update users"},
	{Resource: "user", Action: "delete", Description: "Delete users"},
	{Resource: "user", Action: "import", Description: "Import users"},
	{Resource: "log", Action: "read", Description: "View login logs"},
	{Resource: "log", Action: "delete", Description: "Delete login logs"},
	{Resource: "radius", Action: "read", Description: "View RADIUS billing data"},
	{Resource: "radius", Action: "manage", Description: "Manage RADIUS users, NAS devices, profiles and sessions"},
}
