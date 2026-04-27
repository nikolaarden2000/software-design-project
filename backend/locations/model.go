package locations

type Location struct {
	ID          int     `json:"id"`
	CompanyID   int     `json:"company_id"`
	CompanyName string  `json:"company_name"`
	City        string  `json:"city"`
	Address     string  `json:"address"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Timezone    string  `json:"timezone"`
}

type AdminLocation struct {
	ID          int     `json:"id"`
	CompanyID   int     `json:"company_id"`
	CompanyName string  `json:"company_name"`
	City        string  `json:"city"`
	Address     string  `json:"address"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Timezone    string  `json:"timezone"`
	RoomsCount  int     `json:"rooms_count"`
}
