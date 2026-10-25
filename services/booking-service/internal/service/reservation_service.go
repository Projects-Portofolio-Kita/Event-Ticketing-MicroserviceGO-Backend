package service

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/google/uuid"
)

const reservationTTL = 5 * time.Minute

var (
	ErrSeatsUnavailable = errors.New("beberapa kursi yang dipilih sedang di-lock atau sudah terjual")
	ErrInvalidInput     = errors.New("invalid input")
)

type ReservationService struct {
	repository *repository.ReservationRepository
	notifier   notify.Notifier
}

func NewReservationService(repository *repository.ReservationRepository, notifier notify.Notifier) *ReservationService {
	if notifier == nil {
		notifier = notify.Noop()
	}
	return &ReservationService{repository: repository, notifier: notifier}
}

// emit sends a user-facing notification; never affects the business result.
func (s *ReservationService) emit(msg notify.Message) {
	s.notifier.Send(msg)
}

func (s *ReservationService) CreateReservation(req *dto.CreateReservationRequest) (*model.Reservation, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: userId", ErrInvalidInput)
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("%w: eventId", ErrInvalidInput)
	}

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

	currency := req.Currency
	if currency == "" {
		currency = "IDR"
	}

	reservation := &model.Reservation{
		UserID:      userID,
		EventID:     eventID,
		BookingRef:  generateBookingRef(),
		Status:      model.ReservationLocked,
		ExpiresAt:   time.Now().Add(reservationTTL),
		TotalAmount: total,
		Currency:    currency,
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

func (s *ReservationService) GetReservation(id string) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	return s.repository.FindByID(reservationID)
}

func (s *ReservationService) GetSeatStatus(eventID string) ([]repository.SeatStatus, error) {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid eventId", ErrInvalidInput)
	}

	return s.repository.FindSeatStatusByEvent(id)
}

func (s *ReservationService) ConfirmReservation(id string) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	reservation, err := s.repository.FindByID(reservationID)
	if err != nil {
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

func (s *ReservationService) CancelReservation(id string) (*model.Reservation, error) {
	reservationID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reservation id", ErrInvalidInput)
	}

	reservation, err := s.repository.FindByID(reservationID)
	if err != nil {
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
