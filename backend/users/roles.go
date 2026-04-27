package users

const (
	RoleUser      = "user"
	RoleAdmin     = "admin"
	RoleSuperuser = "superuser"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleUser, RoleAdmin, RoleSuperuser:
		return true
	default:
		return false
	}
}
