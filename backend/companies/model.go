package companies

type Company struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	LocationsCount int    `json:"locations_count"`
}
