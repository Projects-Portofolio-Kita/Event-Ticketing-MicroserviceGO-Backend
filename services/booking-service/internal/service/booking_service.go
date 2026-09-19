package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrSeatsUnavailable    = errors.New("one or more requested seats are no longer available")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationExpired  = errors.New("reservation has expired")
	ErrInvalidStatusChange = errors.New("invalid reservation status for this operation")
)

type BookingService struct {
	repo   *repository.BookingRepository
	locker lock.Locker
	ttl    time.Duration
}

func NewBookingService(repo *repository.BookingRepository, locker lock.Locker, ttl time.Duration) *BookingService {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &BookingService{
		repo:   repo,
		locker: locker,
		ttl:    ttl,
	}
}

// CreateReservation executes atomic multi-seat distributed locking and persists the reservation.
func (s *BookingService) CreateReservation(ctx context.Context, req dto.CreateReservationRequest) (*dto.ReservationResponse, error) {
	if len(req.Seats) == 0 {
		return nil, errors.New("at least one seat must be selected")
	}

	seatUUIDs := make([]uuid.UUID, len(req.Seats))
	for i, seat := range req.Seats {
		seatUUIDs[i] = seat.SeatID
	}

	// 1. PostgreSQL Tier: verify seats are not already CONFIRMED or actively LOCKED
	alreadyBooked, err := s.repo.CheckSeatsAlreadyBooked(ctx, req.EventID, seatUUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to verify seat availability: %w", err)
	}
	if alreadyBooked {
		return nil, ErrSeatsUnavailable
	}

	// 2. Redis Distributed Lock Tier: atomic SET NX EX
	var acquiredSeats []string
	eventIDStr := req.EventID.String()
	userIDStr := req.UserID.String()

	for _, seat := range req.Seats {
		seatIDStr := seat.SeatID.String()
		locked, lockErr := s.locker.LockSeat(ctx, eventIDStr, seatIDStr, userIDStr, s.ttl)
		if lockErr != nil || !locked {
			// Rollback all previously acquired locks in this request batch
			_ = s.locker.ReleaseSeats(ctx, eventIDStr, acquiredSeats)
			return nil, ErrSeatsUnavailable
		}
		acquiredSeats = append(acquiredSeats, seatIDStr)
	}

	// 3. Compute total and construct reservation entity
	var totalAmount float64
	reservationItems := make([]model.ReservationItem, len(req.Seats))
	reservationID := uuid.New()

	for i, seat := range req.Seats {
		totalAmount += seat.Price
		reservationItems[i] = model.ReservationItem{
			ID:            uuid.New(),
			ReservationID: reservationID,
			SeatID:        seat.SeatID,
			Section:       seat.Section,
			Row:           seat.Row,
			Number:        seat.Number,
			Price:         seat.Price,
		}
	}

	currency := req.Currency
	if currency == "" {
		currency = "IDR"
	}

	bookingRef, err := generateBookingRef()
	if err != nil {
		bookingRef = fmt.Sprintf("TKT-%d", time.Now().UnixNano())
	}

	expiresAt := time.Now().Add(s.ttl)

	reservation := &model.Reservation{
		ID:          reservationID,
		UserID:      req.UserID,
		EventID:     req.EventID,
		BookingRef:  bookingRef,
		Status:      model.StatusLocked,
		TotalAmount: totalAmount,
		Currency:    currency,
		ExpiresAt:   expiresAt,
	}

	// 4. Persist in database
	if err := s.repo.CreateWithItems(ctx, reservation, reservationItems); err != nil {
		// Rollback Redis locks if DB write fails
		_ = s.locker.ReleaseSeats(ctx, eventIDStr, acquiredSeats)
		return nil, fmt.Errorf("failed to persist reservation: %w", err)
	}

	return toReservationResponse(reservation), nil
}

// GetReservation retrieves reservation details and updates status if expired.
func (s *BookingService) GetReservation(ctx context.Context, id uuid.UUID) (*dto.ReservationResponse, error) {
	reservation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	// If currently LOCKED but expired, update status and release locks
	if reservation.Status == model.StatusLocked && time.Now().After(reservation.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, reservation.ID, model.StatusExpired)
		reservation.Status = model.StatusExpired

		for _, item := range reservation.Items {
			_ = s.locker.ReleaseSeat(ctx, reservation.EventID.String(), item.SeatID.String())
		}
	}

	return toReservationResponse(reservation), nil
}

// ConfirmReservation marks a reservation as CONFIRMED upon successful payment.
func (s *BookingService) ConfirmReservation(ctx context.Context, id uuid.UUID) (*dto.ReservationResponse, error) {
	reservation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if reservation.Status != model.StatusLocked {
		if reservation.Status == model.StatusConfirmed {
			return toReservationResponse(reservation), nil
		}
		return nil, ErrInvalidStatusChange
	}

	if time.Now().After(reservation.ExpiresAt) {
		_ = s.repo.UpdateStatus(ctx, reservation.ID, model.StatusExpired)
		return nil, ErrReservationExpired
	}

	if err := s.repo.UpdateStatus(ctx, reservation.ID, model.StatusConfirmed); err != nil {
		return nil, fmt.Errorf("failed to confirm reservation: %w", err)
	}

	reservation.Status = model.StatusConfirmed

	// Release Redis locks so permanent database state handles the sold status
	for _, item := range reservation.Items {
		_ = s.locker.ReleaseSeat(ctx, reservation.EventID.String(), item.SeatID.String())
	}

	return toReservationResponse(reservation), nil
}

// CancelReservation releases the held seats and marks reservation CANCELLED.
func (s *BookingService) CancelReservation(ctx context.Context, id uuid.UUID) error {
	reservation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReservationNotFound
		}
		return err
	}

	if reservation.Status != model.StatusLocked {
		return ErrInvalidStatusChange
	}

	if err := s.repo.UpdateStatus(ctx, reservation.ID, model.StatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel reservation: %w", err)
	}

	// Release locks
	for _, item := range reservation.Items {
		_ = s.locker.ReleaseSeat(ctx, reservation.EventID.String(), item.SeatID.String())
	}

	return nil
}

// SweepExpired releases seats from expired reservations in batches.
func (s *BookingService) SweepExpired(ctx context.Context) (int, error) {
	expiredList, err := s.repo.FindExpiredLockedReservations(ctx, 100)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, res := range expiredList {
		if err := s.repo.UpdateStatus(ctx, res.ID, model.StatusExpired); err == nil {
			count++
			for _, item := range res.Items {
				_ = s.locker.ReleaseSeat(ctx, res.EventID.String(), item.SeatID.String())
			}
		}
	}

	return count, nil
}

func toReservationResponse(r *model.Reservation) *dto.ReservationResponse {
	seats := make([]dto.SeatResponseDTO, len(r.Items))
	for i, item := range r.Items {
		seats[i] = dto.SeatResponseDTO{
			SeatID:  item.SeatID,
			Section: item.Section,
			Row:     item.Row,
			Number:  item.Number,
			Price:   item.Price,
		}
	}

	return &dto.ReservationResponse{
		ReservationID: r.ID,
		UserID:        r.UserID,
		EventID:       r.EventID,
		BookingRef:    r.BookingRef,
		Status:        string(r.Status),
		ExpiresAt:     r.ExpiresAt,
		TotalAmount:   r.TotalAmount,
		Currency:      r.Currency,
		Seats:         seats,
		CreatedAt:     r.CreatedAt,
	}
}

func generateBookingRef() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return fmt.Sprintf("TKT-2026-%s", string(b)), nil
}
