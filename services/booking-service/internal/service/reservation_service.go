package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/google/uuid"
)

const (
	reservationTTL = 5 * time.Minute
	// maxTierTickets caps one basket of tier tickets: enough for a group, too
	// small for a reseller to drain a tier in a single request.
	maxTierTickets = 10
)

var (
	ErrSeatsUnavailable   = errors.New("beberapa kursi yang dipilih sedang di-lock atau sudah terjual")
	ErrTicketsUnavailable = errors.New("kuota tiket sudah habis atau tiket tidak tersedia")
	ErrEventNotFound      = errors.New("event tidak ditemukan")
	ErrInvalidInput       = errors.New("invalid input")
	ErrForbidden          = errors.New("Anda tidak punya akses ke reservasi ini.")
)

// reservationStore is the slice of the repository this service depends on.
// Keeping it an interface lets tests substitute an in-memory fake.
type reservationStore interface {
	Create(reservation *model.Reservation) error
	// CreateTier writes a TIER reservation under per-tier advisory locks and
	// asks verify (with fresh reserved counts) whether the quota still fits.
	CreateTier(reservation *model.Reservation, verify func(reserved map[uuid.UUID]int) error) error
	FindByID(id uuid.UUID) (*model.Reservation, error)
	FindByUser(userID uuid.UUID, limit int) ([]model.Reservation, error)
	Update(reservation *model.Reservation) error
	FindConflictingSeats(seatIDs []uuid.UUID) ([]uuid.UUID, error)
	FindSeatStatusByEvent(eventID uuid.UUID) ([]repository.SeatStatus, error)
	FindTicketStatusByEvent(eventID uuid.UUID) ([]repository.TicketStatus, error)
}

type ReservationService struct {
	repository reservationStore
	catalog    client.CatalogClient
	notifier   notify.Notifier
}

// NewReservationService wires the store, the catalog stock feed (used for
// tier prices/quotas) and the notifier. A nil catalog is tolerated so seated
// flows keep working in tests, but tier bookings then fail fast.
func NewReservationService(repository reservationStore, catalog client.CatalogClient, notifier notify.Notifier) *ReservationService {
	if notifier == nil {
		notifier = notify.Noop()
	}
	return &ReservationService{repository: repository, catalog: catalog, notifier: notifier}
}

// ensureOwnership rejects a non-internal caller that is not the owner of the
// given user id. This is the only gate between two users' data.
func ensureOwnership(caller authz.Caller, owner uuid.UUID) error {
	if caller.Internal {
		return nil
	}
	if caller.UserID == "" || caller.UserID != owner.String() {
		return ErrForbidden
	}
	return nil
}

// emit sends a user-facing notification; never affects the business result.
func (s *ReservationService) emit(msg notify.Message) {
	s.notifier.Send(msg)
}

// CreateReservation locks seats (SEATED) or ticket tiers (TIER) for the
// authenticated user. The userId in the body is ignored: identity comes from
// the verified access token (caller).
func (s *ReservationService) CreateReservation(req *dto.CreateReservationRequest, caller authz.Caller) (*model.Reservation, error) {
	userID, err := uuid.Parse(caller.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

	mode, err := resolveMode(req)
	if err != nil {
		return nil, err
	}

	if mode == model.ReservationModeTier {
		return s.createTierReservation(req, userID, eventID)
	}
	return s.createSeatedReservation(req, userID, eventID)
}

// resolveMode picks the booking flow. An explicit request mode wins, a silent
// one follows the payload (items → TIER, otherwise SEATED). Either way the
// chosen mode is checked against its payload, so seat lines and tier lines can
// never be mixed in one reservation.
func resolveMode(req *dto.CreateReservationRequest) (model.ReservationMode, error) {
	requested := strings.ToUpper(strings.TrimSpace(req.Mode))
	switch requested {
	case "", string(model.ReservationModeSeated), string(model.ReservationModeTier):
	default:
		return "", fmt.Errorf("%w: mode harus SEATED atau TIER", ErrInvalidInput)
	}

	if requested == "" {
		if len(req.Items) > 0 {
			return model.ReservationModeTier, nil
		}
		return model.ReservationModeSeated, nil
	}

	mode := model.ReservationMode(requested)
	if mode == model.ReservationModeTier && len(req.Items) == 0 {
		return "", fmt.Errorf("%w: reservasi TIER butuh minimal satu tiket", ErrInvalidInput)
	}
	if mode == model.ReservationModeSeated && len(req.Seats) == 0 {
		return "", fmt.Errorf("%w: minimal satu kursi harus dipilih", ErrInvalidInput)
	}
	return mode, nil
}

// currencyOf defaults the basket currency to IDR.
func currencyOf(currency string) string {
	if currency == "" {
		return "IDR"
	}
	return currency
}

// createSeatedReservation is the original seat-lock flow: reject conflicting
// seats, then persist the lock.
func (s *ReservationService) createSeatedReservation(req *dto.CreateReservationRequest, userID, eventID uuid.UUID) (*model.Reservation, error) {
	if len(req.Seats) == 0 {
		return nil, fmt.Errorf("%w: minimal satu kursi harus dipilih", ErrInvalidInput)
	}

	seatIDs := make([]uuid.UUID, 0, len(req.Seats))
	seats := make([]model.ReservationSeat, 0, len(req.Seats))
	var total float64

	for _, item := range req.Seats {
		seatID, err := uuid.Parse(item.SeatID)
		if err != nil {
			return nil, fmt.Errorf("%w: seatId %s", ErrInvalidInput, item.SeatID)
		}

		seatIDs = append(seatIDs, seatID)
		total += item.Price

		seats = append(seats, model.ReservationSeat{
			SeatID:  seatID,
			Section: item.Section,
			Row:     item.Row,
			Number:  item.Number,
			Price:   item.Price,
		})
	}

	conflicts, err := s.repository.FindConflictingSeats(seatIDs)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, ErrSeatsUnavailable
	}

	reservation := &model.Reservation{
		UserID:      userID,
		EventID:     eventID,
		BookingRef:  generateBookingRef(),
		Status:      model.ReservationLocked,
		Mode:        model.ReservationModeSeated,
		ExpiresAt:   time.Now().Add(reservationTTL),
		TotalAmount: total,
		Currency:    currencyOf(req.Currency),
		Seats:       seats,
	}

	if err := s.repository.Create(reservation); err != nil {
		return nil, err
	}

	s.emit(notify.Message{
		UserID:    userID.String(),
		Kind:      notify.KindReservationLocked,
		Title:     "Kursi sedang dikunci",
		Body:      fmt.Sprintf("Reservasi %s berhasil dibuat, kursi dikunci sampai %s. Selesaikan pembayaran agar tidak lepas.", reservation.BookingRef, reservation.ExpiresAt.Local().Format("15:04")),
		Reference: reservation.ID.String(),
	})

	return reservation, nil
}

// createTierReservation locks quota-based tickets (VIP/VVIP/Regular or custom
// tiers). It never trusts the client for money or stock: prices, tier names
// and quotas come from catalog-service, and the reserved count is re-read
// inside CreateTier's advisory-locked transaction before the quota check.
func (s *ReservationService) createTierReservation(req *dto.CreateReservationRequest, userID, eventID uuid.UUID) (*model.Reservation, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: minimal satu tiket harus dipilih", ErrInvalidInput)
	}
	if s.catalog == nil {
		return nil, fmt.Errorf("%w: katalog tiket tidak terhubung", ErrTicketsUnavailable)
	}

	// Merge duplicate lines (2 x VIP + 1 x VIP = 3 x VIP) and cap the basket.
	quantities := make(map[uuid.UUID]int, len(req.Items))
	order := make([]uuid.UUID, 0, len(req.Items))
	totalQuantity := 0
	for _, item := range req.Items {
		categoryID, err := uuid.Parse(item.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("%w: categoryId %s", ErrInvalidInput, item.CategoryID)
		}
		if item.Quantity < 1 || item.Quantity > maxTierTickets {
			return nil, fmt.Errorf("%w: jumlah tiket harus antara 1 dan %d", ErrInvalidInput, maxTierTickets)
		}
		if _, seen := quantities[categoryID]; !seen {
			order = append(order, categoryID)
		}
		quantities[categoryID] += item.Quantity
		totalQuantity += item.Quantity
		if totalQuantity > maxTierTickets {
			return nil, fmt.Errorf("%w: maksimal %d tiket per reservasi", ErrInvalidInput, maxTierTickets)
		}
	}

	tiers, err := s.catalog.GetTierQuotas(eventID.String())
	if err != nil {
		if errors.Is(err, client.ErrEventNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	byID := make(map[uuid.UUID]client.TierQuota, len(tiers))
	for _, tier := range tiers {
		categoryID, err := uuid.Parse(tier.CategoryID)
		if err != nil {
			continue
		}
		byID[categoryID] = tier
	}

	items := make([]model.ReservationItem, 0, len(order))
	var total float64
	for _, categoryID := range order {
		tier, ok := byID[categoryID]
		if !ok {
			return nil, fmt.Errorf("%w: tier %s tidak dijual di event ini", ErrTicketsUnavailable, categoryID)
		}

		quantity := quantities[categoryID]
		subtotal := tier.Price * float64(quantity)
		total += subtotal

		items = append(items, model.ReservationItem{
			CategoryID:   categoryID,
			CategoryName: tier.Name,
			Quantity:     quantity,
			UnitPrice:    tier.Price,
			Subtotal:     subtotal,
		})
	}

	reservation := &model.Reservation{
		UserID:      userID,
		EventID:     eventID,
		BookingRef:  generateBookingRef(),
		Status:      model.ReservationLocked,
		Mode:        model.ReservationModeTier,
		ExpiresAt:   time.Now().Add(reservationTTL),
		TotalAmount: total,
		Currency:    currencyOf(req.Currency),
		Items:       items,
	}

	// verify runs inside the locked transaction with freshly counted stock.
	verify := func(reserved map[uuid.UUID]int) error {
		for _, item := range items {
			quota := byID[item.CategoryID].Quota
			if used := reserved[item.CategoryID]; used+item.Quantity > quota {
				return fmt.Errorf("%w: %s tersisa %d", ErrTicketsUnavailable, item.CategoryName, quota-used)
			}
		}
		return nil
	}

	if err := s.repository.CreateTier(reservation, verify); err != nil {
		return nil, err
	}

	s.emit(notify.Message{
		UserID:    userID.String(),
		Kind:      notify.KindReservationLocked,
		Title:     "Tiket sedang dikunci",
		Body:      fmt.Sprintf("Reservasi %s (%d tiket) berhasil dibuat, tiket dikunci sampai %s. Selesaikan pembayaran agar tidak lepas.", reservation.BookingRef, totalQuantity, reservation.ExpiresAt.Local().Format("15:04")),
		Reference: reservation.ID.String(),
	})

	return reservation, nil
}

func (s *ReservationService) GetReservation(id string, caller authz.Caller) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	reservation, err := s.repository.FindByID(reservationID)
	if err != nil {
		return nil, err
	}

	if err := ensureOwnership(caller, reservation.UserID); err != nil {
		return nil, err
	}

	return reservation, nil
}

func (s *ReservationService) GetSeatStatus(eventID string) ([]repository.SeatStatus, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid eventId", ErrInvalidInput)
	}

	return s.repository.FindSeatStatusByEvent(id)
}

// ListReservations returns the caller's own reservations (newest first) — the
// data behind "Tiket Saya". Identity always comes from the token.
func (s *ReservationService) ListReservations(caller authz.Caller, limit int) ([]model.Reservation, error) {
	userID, err := uuid.Parse(caller.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}

	return s.repository.FindByUser(userID, limit)
}

// GetEventTicketStatus is the tier availability feed catalog-service reads
// (X-Service-Token): how many tickets of each tier are locked or sold.
func (s *ReservationService) GetEventTicketStatus(eventID string) ([]repository.TicketStatus, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid eventId", ErrInvalidInput)
	}

	return s.repository.FindTicketStatusByEvent(id)
}

func (s *ReservationService) ConfirmReservation(id string, caller authz.Caller) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	reservation, err := s.repository.FindByID(reservationID)
	if err != nil {
		return nil, err
	}

	if err := ensureOwnership(caller, reservation.UserID); err != nil {
		return nil, err
	}

	if reservation.Status != model.ReservationLocked {
		return nil, fmt.Errorf("%w: reservasi tidak dalam status LOCKED", ErrInvalidInput)
	}

	if time.Now().After(reservation.ExpiresAt) {
		reservation.Status = model.ReservationExpired
		_ = s.repository.Update(reservation)
		return nil, fmt.Errorf("%w: reservasi sudah kedaluwarsa", ErrInvalidInput)
	}

	reservation.Status = model.ReservationConfirmed
	if err := s.repository.Update(reservation); err != nil {
		return nil, err
	}

	s.emit(notify.Message{
		UserID:    reservation.UserID.String(),
		Kind:      notify.KindBookingConfirmed,
		Title:     "Tiket dikonfirmasi",
		Body:      fmt.Sprintf("Reservasi %s berstatus CONFIRMED. Total %s %.0f.", reservation.BookingRef, reservation.Currency, reservation.TotalAmount),
		Reference: reservation.ID.String(),
	})

	return reservation, nil
}

func (s *ReservationService) CancelReservation(id string, caller authz.Caller) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	reservation, err := s.repository.FindByID(reservationID)
	if err != nil {
		return nil, err
	}

	if err := ensureOwnership(caller, reservation.UserID); err != nil {
		return nil, err
	}

	if reservation.Status == model.ReservationConfirmed {
		return nil, fmt.Errorf("%w: reservasi yang sudah dikonfirmasi tidak bisa dibatalkan", ErrInvalidInput)
	}

	reservation.Status = model.ReservationCancelled
	if err := s.repository.Update(reservation); err != nil {
		return nil, err
	}

	s.emit(notify.Message{
		UserID:    reservation.UserID.String(),
		Kind:      notify.KindBookingCancelled,
		Title:     "Reservasi dibatalkan",
		Body:      fmt.Sprintf("Reservasi %s dibatalkan dan kursi dilepas kembali.", reservation.BookingRef),
		Reference: reservation.ID.String(),
	})

	return reservation, nil
}

func generateBookingRef() string {
	return fmt.Sprintf("TICK-%d-%04d", time.Now().Unix(), rand.Intn(10000))
}
