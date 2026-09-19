package worker

import (
	"context"
	"log"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
)

type Sweeper struct {
	bookingService *service.BookingService
	interval       time.Duration
}

func NewSweeper(bookingService *service.BookingService, interval time.Duration) *Sweeper {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Sweeper{
		bookingService: bookingService,
		interval:       interval,
	}
}

// Start begins the background sweeper loop. It stops when the context is cancelled.
func (s *Sweeper) Start(ctx context.Context) {
	log.Printf("[INFO] Background sweeper started with interval: %v", s.interval)
	ticker := time.NewTicker(s.interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count, err := s.bookingService.SweepExpired(ctx)
				if err != nil {
					log.Printf("[ERROR] Sweeper error checking expired reservations: %v", err)
				} else if count > 0 {
					log.Printf("[INFO] Sweeper auto-released %d expired reservation(s)", count)
				}
			case <-ctx.Done():
				log.Println("[INFO] Background sweeper stopped.")
				return
			}
		}
	}()
}
