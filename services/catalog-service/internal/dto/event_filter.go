package dto

// EventFilter carries the discovery query of GET /events. Every field is
// optional; zero values mean "no restriction".
type EventFilter struct {
	// Q matches name, venue and description (case-insensitive).
	Q string `form:"q"`
	// City matches the event's location exactly (case-insensitive).
	City string `form:"city"`
	// Category is one of the event categories (concert, festival, ...).
	Category string `form:"category"`
	// Mode is SEATED or TIER.
	Mode string `form:"mode"`
	// CreatedBy limits the list to one poster's events (uuid).
	CreatedBy string `form:"createdBy"`
	// Mine is a shortcut for CreatedBy = the caller (set by the handler).
	Mine bool `form:"mine"`
	// Page/Limit paginate; zero means defaults (1 / 50).
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// Normalized defaults so the repository never sees a zero page/limit.
func (f EventFilter) Normalized() (page, limit int) {
	page = f.Page
	if page < 1 {
		page = 1
	}
	limit = f.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
