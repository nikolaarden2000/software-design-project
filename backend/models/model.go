package models

type Room struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Address  string `json:"address"`
	Capacity int    `json:"capacity"`
	ImageURL string `json:"image"`
	Price    int    `json:"price"`
}

type User struct {
	ID       int
	Username string
	Email    string
	Password string
	Role     string
}

type RoomPageData struct {
	ID                int
	Title             string
	Company           string
	Address           string
	Images            []string
	Price             int
	Currency          string
	Capacity          int
	AvailableFrom     string
	AvailableTo       string
	Description       string
	DescriptionHTML   string
	DescriptionIsHTML bool
	MaxCapacity       int
	Lat               float64
	Lng               float64
}

type DateAvailability struct {
	Date           string   `json:"date"`
	AvailableTimes []string `json:"available_times"`
}

type BookingHistoryItem struct {
	ID         int    `json:"id"`
	RoomID     int    `json:"room_id"`
	ImageURL   string `json:"image_url"`
	Title      string `json:"title"`
	Date       string `json:"date"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	TotalPrice int    `json:"total_price"`
	Status     string `json:"status"`
}
