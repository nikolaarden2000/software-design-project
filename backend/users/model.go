package users

type User struct {
	ID       int
	Username string
	Email    string
	Password string
	Role     string
}

type AdminLocation struct {
	ID          int    `json:"id"`
	Address     string `json:"address"`
	CompanyName string `json:"company_name"`
}

type Admin struct {
	ID        int             `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      string          `json:"role"`
	Locations []AdminLocation `json:"locations"`
}
