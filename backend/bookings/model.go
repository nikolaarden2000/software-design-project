package bookings

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
