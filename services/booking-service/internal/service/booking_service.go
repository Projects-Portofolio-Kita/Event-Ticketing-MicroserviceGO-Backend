// Package service coordinates booking business logic, locking, and lifecycle management.
//
// File: booking_service.go
// Usability: Declares core BookingService struct, sentinel domain errors, and constructor.
package service

import (
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/lock"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
)

var (
	ErrSeatsUnavailable    = errors.New("one or more requested seats are no longer available")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationExpired  = errors.New("reservation has expired")
	ErrInvalidStatusChange = errors.New("invalid reservation status for this operation")
)

// BookingService manages seat reservation lifecycle, locking, and status transitions.
type BookingService struct {
	repo   *repository.BookingRepository
	locker lock.Locker
	ttl    time.Duration
}

// NewBookingService initializes a new BookingService with default 5-minute TTL.
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
