package router

// Auth route contracts are kept here so authentication paths and permissions are
// not duplicated across handlers, clients, and route registration.
const authRoutePrefix = "/auth"

const (
	authPermissionResource = "auth"
	authPermissionRead     = "read"
)

func authRoute(path string) string {
	return authRoutePrefix + path
}
