package rooms

type Room struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Address  string `json:"address"`
	Capacity int    `json:"capacity"`
	ImageURL string `json:"image"`
	Price    int    `json:"price"`
}

type RoomPageData struct {
	ID                int      `json:"id"`
	Title             string   `json:"title"`
	Company           string   `json:"company"`
	Address           string   `json:"address"`
	Images            []string `json:"images"`
	Price             int      `json:"price"`
	Currency          string   `json:"currency"`
	Capacity          int      `json:"capacity"`
	AvailableFrom     string   `json:"available_from"`
	AvailableTo       string   `json:"available_to"`
	Description       string   `json:"description"`
	DescriptionHTML   string   `json:"description_html,omitempty"`
	DescriptionIsHTML bool     `json:"description_is_html,omitempty"`
	MaxCapacity       int      `json:"max_capacity"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
}

type DateAvailability struct {
	Date           string   `json:"date"`
	AvailableTimes []string `json:"available_times"`
}
