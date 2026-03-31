package handler

// Route constants
const (
	RouteHome     = "/"
	RouteLogin    = "/auth/login"
	RouteRegister = "/auth/register"
	RouteLogout   = "/auth/logout"
	RouteProfile  = "/profile"

	RouteAPIAuthState    = "/api/auth/state"
	RouteAPIAuthLogin    = "/api/auth/login"
	RouteAPIAuthRegister = "/api/auth/register"
	RouteAPIAuthLogout   = "/api/auth/logout"
)

// Response messages
const (
	MsgLoginSuccess          = "Login successful"
	MsgLogoutSuccess         = "Logout successful"
	MsgRegisterSuccess       = "Registration successful"
	MsgUserCreateSuccess     = "User created successfully"
	MsgUserUpdateSuccess     = "User updated successfully"
	MsgUserDeactivateSuccess = "User deactivated successfully"
	MsgUserDeleteSuccess     = "User deleted successfully"
)
