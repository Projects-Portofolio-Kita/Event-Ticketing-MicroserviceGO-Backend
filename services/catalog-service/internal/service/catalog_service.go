package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/catalog-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidInput  = errors.New("invalid input")
)

// eventStore is the consumer-side slice of repository.CatalogRepository the
// service actually uses. Depending on this narrow interface keeps the service
// testable with an in-memory fake; *repository.CatalogRepository satisfies it.
type eventStore interface {
	FindEvents(filter dto.EventFilter) ([]model.Event, error)
	FindEventCount(filter dto.EventFilter) (int64, error)
	FindDistinctCities() ([]dto.CityEntry, error)
	FindByID(id uuid.UUID) (*model.Event, error)
	FindCategoriesByEvent(eventID uuid.UUID) ([]model.SeatCategory, error)
	FindCategoriesByEvents(eventIDs []uuid.UUID) ([]model.SeatCategory, error)
	FindSeatsByEvent(eventID uuid.UUID) ([]model.Seat, error)
	CodeExists(code string) (bool, error)
	CreateEventWithSeatmap(event *model.Event, categories []model.SeatCategory, seats []model.Seat) error
}

type CatalogService struct {
	repository eventStore
	booking    client.BookingClient
}

func NewCatalogService(repository eventStore, booking client.BookingClient) *CatalogService {
	return &CatalogService{
		repository: repository,
		booking:    booking,
	}
}

// ListEvents returns one page of events plus the unpaginated total, enriched
// with minPrice / ticketTypeCount from the categories of the listed events.
func (s *CatalogService) ListEvents(filter dto.EventFilter) ([]dto.EventResponse, int64, error) {
	events, err := s.repository.FindEvents(filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repository.FindEventCount(filter)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}

	categories, err := s.repository.FindCategoriesByEvents(ids)
	if err != nil {
		return nil, 0, err
	}

	byEvent := map[uuid.UUID][]model.SeatCategory{}
	for _, category := range categories {
		byEvent[category.EventID] = append(byEvent[category.EventID], category)
	}

	result := make([]dto.EventResponse, 0, len(events))
	for _, event := range events {
		result = append(result, toEventResponse(event, byEvent[event.ID]))
	}

	return result, total, nil
}

// toEventResponse maps one event plus its categories onto the discovery shape.
func toEventResponse(event model.Event, categories []model.SeatCategory) dto.EventResponse {
	var minPrice float64
	if len(categories) > 0 {
		minPrice = categories[0].Price
		for _, category := range categories {
			if category.Price < minPrice {
				minPrice = category.Price
			}
		}
	}

	createdBy := ""
	if event.CreatedBy != nil {
		createdBy = event.CreatedBy.String()
	}

	return dto.EventResponse{
		ID:              event.ID.String(),
		Code:            event.Code,
		Name:            event.Name,
		Venue:           event.Venue,
		EventDate:       event.EventDate.Format(time.RFC3339),
		City:            event.City,
		Category:        event.Category,
		SaleMode:        string(event.SaleMode),
		Description:     event.Description,
		PosterURL:       event.PosterURL,
		MinPrice:        minPrice,
		TicketTypeCount: len(categories),
		CreatedBy:       createdBy,
	}
}

// GetEvent returns one event by id (detail view).
func (s *CatalogService) GetEvent(eventID string) (*dto.EventResponse, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	event, err := s.repository.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	categories, err := s.repository.FindCategoriesByEvent(id)
	if err != nil {
		return nil, err
	}

	response := toEventResponse(*event, categories)
	return &response, nil
}

// ListCities feeds the location picker dropdown.
func (s *CatalogService) ListCities() ([]dto.CityEntry, error) {
	return s.repository.FindDistinctCities()
}

// GetTicketTypes returns a TIER event's tiers with live remaining stock.
// booking-service owns availability; when it is down remaining degrades to 0
// rather than failing the request.
func (s *CatalogService) GetTicketTypes(eventID string) (*dto.TicketTypeListResponse, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	event, err := s.repository.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	categories, err := s.repository.FindCategoriesByEvent(id)
	if err != nil {
		return nil, err
	}

	reserved := map[string]int{}
	if statuses, err := s.booking.GetTicketStatus(eventID); err == nil {
		for _, status := range statuses {
			reserved[status.CategoryID] = status.Reserved
		}
	}

	ticketTypes := make([]dto.TicketTypeResponse, 0, len(categories))
	for _, category := range categories {
		remaining := category.Quota - reserved[category.ID.String()]
		if remaining < 0 {
			remaining = 0
		}
		ticketTypes = append(ticketTypes, dto.TicketTypeResponse{
			CategoryID:  category.ID.String(),
			Name:        category.Name,
			Description: category.Description,
			Price:       category.Price,
			Quota:       category.Quota,
			Remaining:   remaining,
		})
	}

	return &dto.TicketTypeListResponse{
		SaleMode:    string(event.SaleMode),
		TicketTypes: ticketTypes,
	}, nil
}

// GetTierQuotas is the internal stock feed booking-service reads when it
// validates a tier reservation. It is deliberately static (no availability):
// booking-service computes availability from its own reservations.
func (s *CatalogService) GetTierQuotas(eventID string) (*dto.TierQuotaListResponse, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	if _, err := s.repository.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	categories, err := s.repository.FindCategoriesByEvent(id)
	if err != nil {
		return nil, err
	}

	tiers := make([]dto.TierQuota, 0, len(categories))
	for _, category := range categories {
		tiers = append(tiers, dto.TierQuota{
			CategoryID: category.ID.String(),
			Name:       category.Name,
			Price:      category.Price,
			Quota:      category.Quota,
		})
	}

	return &dto.TierQuotaListResponse{Tiers: tiers}, nil
}

func (s *CatalogService) GetSeatMap(eventID string) ([]dto.CategoryResponse, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	if _, err := s.repository.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	categories, err := s.repository.FindCategoriesByEvent(id)
	if err != nil {
		return nil, err
	}

	seats, err := s.repository.FindSeatsByEvent(id)
	if err != nil {
		return nil, err
	}

	// Seat availability is owned by booking-service; degrade to "available" when it is down.
	statuses := map[string]string{}
	if liveStatuses, err := s.booking.GetSeatStatus(eventID); err == nil {
		for _, status := range liveStatuses {
			statuses[status.SeatID] = status.Status
		}
	}

	seatsByCategory := map[uuid.UUID][]model.Seat{}
	for _, seat := range seats {
		seatsByCategory[seat.CategoryID] = append(seatsByCategory[seat.CategoryID], seat)
	}

	result := make([]dto.CategoryResponse, 0, len(categories))
	for _, category := range categories {
		rows := buildRows(seatsByCategory[category.ID], category, statuses)

		result = append(result, dto.CategoryResponse{
			ID:       category.Code,
			Name:     category.Name,
			Subtitle: category.Subtitle,
			DotColor: category.DotColor,
			Price:    category.Price,
			Rows:     rows,
		})
	}

	return result, nil
}

func buildRows(categorySeats []model.Seat, category model.SeatCategory, statuses map[string]string) []dto.RowResponse {
	rowOrder := []string{}
	rowsByName := map[string]*dto.RowResponse{}

	for _, seat := range categorySeats {
		row, exists := rowsByName[seat.RowName]
		if !exists {
			row = &dto.RowResponse{
				RowName:    seat.RowName,
				LeftSeats:  []dto.SeatResponse{},
				RightSeats: []dto.SeatResponse{},
			}
			rowsByName[seat.RowName] = row
			rowOrder = append(rowOrder, seat.RowName)
		}

		item := dto.SeatResponse{
			ID:       seat.Label,
			SeatID:   seat.ID.String(),
			Row:      seat.RowName,
			Number:   seat.Number,
			Status:   resolveStatus(statuses[seat.ID.String()]),
			Category: category.Name,
			Price:    category.Price,
		}

		if seat.Side == "right" {
			row.RightSeats = append(row.RightSeats, item)
		} else {
			row.LeftSeats = append(row.LeftSeats, item)
		}
	}

	rows := make([]dto.RowResponse, 0, len(rowOrder))
	for _, rowName := range rowOrder {
		rows = append(rows, *rowsByName[rowName])
	}

	return rows
}

func resolveStatus(bookingStatus string) string {
	switch bookingStatus {
	case "CONFIRMED":
		return "sold"
	case "LOCKED":
		return "locked"
	default:
		return "available"
	}
}

// Layout of the automatically generated seatmap: rows A-D, seats 1-8 on the
// left, 9-16 on the right (mirrors the Coldplay seed layout).
const (
	defaultRows        = 4
	defaultSeatsPerRow = 16
	defaultLeftBlock   = 8
	defaultCity        = "Jakarta"
	defaultCategory    = "concert"
	genMaxRetries      = 50
	genMaxCodeLen      = 44
)

var seatRowNames = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

// CreateEvent posts a new event on behalf of the caller: the event row, its
// priced categories and a ready-to-book seatmap are persisted in one
// transaction. Identity comes exclusively from the access token (caller), so
// a client cannot post as someone else.
//
// SaleMode "SEATED" generates a numbered seat map; "TIER" sells quota-based
// tickets (VIP/VVIP/Regular or any custom tier) and generates no seats.
func (s *CatalogService) CreateEvent(req *dto.CreateEventRequest, caller authz.Caller) (*model.Event, error) {
	name := strings.TrimSpace(req.Name)
	venue := strings.TrimSpace(req.Venue)
	if name == "" {
		return nil, fmt.Errorf("%w: nama event wajib diisi", ErrInvalidInput)
	}
	if venue == "" {
		return nil, fmt.Errorf("%w: venue wajib diisi", ErrInvalidInput)
	}
	if len(req.Categories) == 0 {
		return nil, fmt.Errorf("%w: minimal satu kategori kursi", ErrInvalidInput)
	}

	city := strings.TrimSpace(req.City)
	if city == "" {
		city = defaultCity
	}
	if len(city) > 100 {
		return nil, fmt.Errorf("%w: kota maksimal 100 karakter", ErrInvalidInput)
	}

	category := strings.ToLower(strings.TrimSpace(req.Category))
	if category == "" {
		category = defaultCategory
	}
	if !model.ValidEventCategories[category] {
		return nil, fmt.Errorf("%w: kategori event %q tidak dikenal", ErrInvalidInput, category)
	}

	description := strings.TrimSpace(req.Description)
	if len(description) > 2000 {
		return nil, fmt.Errorf("%w: deskripsi maksimal 2000 karakter", ErrInvalidInput)
	}

	posterURL := strings.TrimSpace(req.PosterURL)
	if posterURL != "" && !strings.HasPrefix(posterURL, "http://") && !strings.HasPrefix(posterURL, "https://") {
		return nil, fmt.Errorf("%w: posterUrl harus http(s)", ErrInvalidInput)
	}

	saleMode, err := resolveSaleMode(req)
	if err != nil {
		return nil, err
	}

	seatRows := req.SeatRows
	if seatRows == 0 {
		seatRows = defaultRows
	}
	seatsPerRow := req.SeatsPerRow
	if seatsPerRow == 0 {
		seatsPerRow = defaultSeatsPerRow
	}
	leftBlockSize := req.LeftBlockSize
	if leftBlockSize == 0 {
		leftBlockSize = defaultLeftBlock
	}
	if saleMode == model.SaleModeSeated && leftBlockSize >= seatsPerRow {
		return nil, fmt.Errorf("%w: leftBlockSize harus lebih kecil dari seatsPerRow", ErrInvalidInput)
	}

	payoutBankName := strings.TrimSpace(req.PayoutBankName)
	payoutAccountNumber := strings.TrimSpace(req.PayoutAccountNumber)
	payoutAccountName := strings.TrimSpace(req.PayoutAccountName)
	filledPayoutFields := 0
	if payoutBankName != "" {
		filledPayoutFields++
	}
	if payoutAccountNumber != "" {
		filledPayoutFields++
	}
	if payoutAccountName != "" {
		filledPayoutFields++
	}
	if filledPayoutFields > 0 && filledPayoutFields < 3 {
		return nil, fmt.Errorf("%w: data rekening payout harus lengkap (bank, no rekening, nama pemilik)", ErrInvalidInput)
	}

	eventDate, err := parseEventDate(req.EventDate)
	if err != nil {
		return nil, err
	}

	createdBy, err := uuid.Parse(caller.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: subject token bukan uuid", ErrInvalidInput)
	}

	code, err := s.resolveCode(req.Code, name)
	if err != nil {
		return nil, err
	}

	event := &model.Event{
		ID:                  uuid.New(),
		Code:                code,
		Name:                name,
		Venue:               venue,
		EventDate:           eventDate,
		City:                city,
		Category:            category,
		Description:         description,
		PosterURL:           posterURL,
		SaleMode:            saleMode,
		SeatRows:            seatRows,
		SeatsPerRow:         seatsPerRow,
		LeftBlockSize:       leftBlockSize,
		PayoutBankName:      payoutBankName,
		PayoutAccountNumber: payoutAccountNumber,
		PayoutAccountName:   payoutAccountName,
		CreatedBy:           &createdBy,
	}

	categories := make([]model.SeatCategory, 0, len(req.Categories))
	seats := make([]model.Seat, 0, len(req.Categories)*seatRows*seatsPerRow)

	for i, input := range req.Categories {
		categoryName := strings.TrimSpace(input.Name)
		if categoryName == "" {
			return nil, fmt.Errorf("%w: nama kategori wajib diisi", ErrInvalidInput)
		}
		if input.Price < 0 {
			return nil, fmt.Errorf("%w: harga kategori tidak boleh negatif", ErrInvalidInput)
		}

		quota := input.Quota
		if saleMode == model.SaleModeTier {
			if quota < 1 {
				return nil, fmt.Errorf("%w: tiket %q butuh kuota minimal 1", ErrInvalidInput, categoryName)
			}
		} else {
			// Seated events count availability per seat, not per tier.
			quota = 0
		}

		category := model.SeatCategory{
			ID:          uuid.New(),
			EventID:     event.ID,
			Code:        fmt.Sprintf("%s-%d", code, i+1),
			Name:        categoryName,
			Subtitle:    strings.TrimSpace(input.Description),
			DotColor:    tierColors[i%len(tierColors)],
			Price:       input.Price,
			SortOrder:   i + 1,
			Quota:       quota,
			Description: strings.TrimSpace(input.Description),
		}
		categories = append(categories, category)

		if saleMode == model.SaleModeSeated {
			seats = append(seats, generateSeats(event.ID, category.ID, fmt.Sprintf("C%d", i+1), seatRows, seatsPerRow, leftBlockSize)...)
		}
	}

	if err := s.repository.CreateEventWithSeatmap(event, categories, seats); err != nil {
		return nil, err
	}

	return event, nil
}

// tierColors cycle through the category dots shown in the seat map / tier list.
var tierColors = []string{"#f59e0b", "#8b5cf6", "#10b981", "#0ea5e9", "#ef4444", "#ec4899", "#14b8a6", "#6366f1", "#84cc16", "#f97316"}

// resolveSaleMode picks the sale mode: an explicit request wins, otherwise a
// request carrying quotas becomes TIER and everything else stays SEATED.
func resolveSaleMode(req *dto.CreateEventRequest) (model.SaleMode, error) {
	requested := strings.ToUpper(strings.TrimSpace(req.SaleMode))
	switch requested {
	case "", string(model.SaleModeSeated), string(model.SaleModeTier):
	default:
		return "", fmt.Errorf("%w: saleMode harus SEATED atau TIER", ErrInvalidInput)
	}

	hasQuota := false
	for _, category := range req.Categories {
		if category.Quota > 0 {
			hasQuota = true
			break
		}
	}

	if requested == "" {
		if hasQuota {
			return model.SaleModeTier, nil
		}
		return model.SaleModeSeated, nil
	}

	mode := model.SaleMode(requested)
	if mode == model.SaleModeSeated && hasQuota {
		return "", fmt.Errorf("%w: event SEATED tidak boleh punya kuota tiket", ErrInvalidInput)
	}
	return mode, nil
}

// generateSeats builds rows A-D x seats 1-16 for one category, mirroring the
// seed layout so booking and the seatmap UI work unchanged.
func generateSeats(eventID, categoryID uuid.UUID, prefix string, rows, seatsPerRow, leftBlockSize int) []model.Seat {
	seats := make([]model.Seat, 0, rows*seatsPerRow)

	for rowIndex := 0; rowIndex < rows; rowIndex++ {
		row := seatRowNames[rowIndex]
		for number := 1; number <= seatsPerRow; number++ {
			side := "left"
			if number > leftBlockSize {
				side = "right"
			}
			seats = append(seats, model.Seat{
				EventID:    eventID,
				CategoryID: categoryID,
				Label:      fmt.Sprintf("%s-%s%02d", prefix, row, number),
				RowName:    row,
				Number:     number,
				Side:       side,
				SortOrder:  rowIndex*100 + number,
			})
		}
	}

	return seats
}

// resolveCode returns a unique event code: an explicitly requested code must
// be free (duplicate = invalid input), while an empty one is derived from the
// name and gets a "-2", "-3", ... suffix until free.
func (s *CatalogService) resolveCode(requested, name string) (string, error) {
	base := slugify(requested)
	explicit := base != ""
	if !explicit {
		base = slugify(name)
	}
	if base == "" {
		base = "event"
	}

	candidate := base
	for attempt := 2; ; attempt++ {
		used, err := s.repository.CodeExists(candidate)
		if err != nil {
			return "", err
		}
		if !used {
			return candidate, nil
		}
		if explicit || attempt > genMaxRetries {
			return "", fmt.Errorf("%w: kode event %q sudah dipakai", ErrInvalidInput, candidate)
		}
		candidate = fmt.Sprintf("%s-%d", base, attempt)
	}
}

// slugify lowers the text and keeps only a-z, 0-9 and single hyphens, capped
// so code + "-50" still fits the varchar(50) column.
func slugify(raw string) string {
	var b strings.Builder
	lastHyphen := false

	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen && b.Len() > 0:
			b.WriteRune('-')
			lastHyphen = true
		}
	}

	code := strings.Trim(b.String(), "-")
	if len(code) > genMaxCodeLen {
		code = strings.Trim(code[:genMaxCodeLen], "-")
	}
	return code
}

// parseEventDate accepts RFC3339 ("2026-06-15T19:30:00+07:00") or a plain
// date ("2026-06-15").
func parseEventDate(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%w: eventDate harus RFC3339 atau YYYY-MM-DD", ErrInvalidInput)
}
