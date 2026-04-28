package rooms

const (
	StatusDraft     = "draft"
	StatusPending   = "pending"
	StatusPublished = "published"
	StatusRejected  = "rejected"
	StatusArchived  = "archived"
)

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

type AdminRoomListItem struct {
	ID              int     `json:"id"`
	LocationID      int     `json:"location_id"`
	Title           string  `json:"title"`
	Price           int     `json:"price"`
	Capacity        int     `json:"capacity"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
	CreatedAt       string  `json:"created_at"`
}

type AdminRoomDetails struct {
	ID              int              `json:"id"`
	LocationID      int              `json:"location_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Price           int              `json:"price"`
	Capacity        int              `json:"capacity"`
	AvailableFrom   string           `json:"available_from"`
	AvailableTo     string           `json:"available_to"`
	Images          []string         `json:"images"`
	Status          string           `json:"status"`
	RejectionReason *string          `json:"rejection_reason"`
	Archive         AdminRoomArchive `json:"archive"`
}

type AdminRoomArchive struct {
	CanArchiveNow             bool    `json:"can_archive_now"`
	HasActiveOrFutureBookings bool    `json:"has_active_or_future_bookings"`
	BookingDisabled           bool    `json:"booking_disabled"`
	ScheduledFor              *string `json:"scheduled_for"`
}

const (
	ArchiveModeImmediate = "immediate"
	ArchiveModeScheduled = "scheduled"
)

type AdminRoomArchiveResult struct {
	ID                  int     `json:"id"`
	Status              string  `json:"status"`
	BookingDisabled     bool    `json:"booking_disabled,omitempty"`
	ArchiveScheduledFor *string `json:"archive_scheduled_for,omitempty"`
}

type AdminRoomInput struct {
	LocationID    int      `json:"location_id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Price         int      `json:"price"`
	Capacity      int      `json:"capacity"`
	AvailableFrom string   `json:"available_from"`
	AvailableTo   string   `json:"available_to"`
	Images        []string `json:"images"`
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusDraft, StatusPending, StatusPublished, StatusRejected, StatusArchived:
		return true
	default:
		return false
	}
}

type ModerationRoomCreator struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type ModerationRoom struct {
	ID            int                    `json:"id"`
	LocationID    int                    `json:"location_id"`
	CompanyName   string                 `json:"company_name"`
	City          string                 `json:"city"`
	Address       string                 `json:"address"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Price         int                    `json:"price"`
	Capacity      int                    `json:"capacity"`
	AvailableFrom string                 `json:"available_from"`
	AvailableTo   string                 `json:"available_to"`
	Images        []string               `json:"images"`
	Status        string                 `json:"status"`
	CreatedBy     *ModerationRoomCreator `json:"created_by"`
}
