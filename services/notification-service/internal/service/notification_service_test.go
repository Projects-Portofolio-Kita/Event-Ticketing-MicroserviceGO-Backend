package service

import (
	"errors"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeStore is an in-memory stand-in for NotificationRepository.
type fakeStore struct {
	items map[uuid.UUID]*model.Notification
	order []uuid.UUID
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[uuid.UUID]*model.Notification{}}
}

func (f *fakeStore) Create(notification *model.Notification) error {
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	f.items[notification.ID] = notification
	f.order = append(f.order, notification.ID)
	return nil
}

func (f *fakeStore) ListByUser(userID uuid.UUID, limit int) ([]model.Notification, error) {
	if limit <= 0 {
		limit = 20
	}

	var out []model.Notification
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		item := f.items[f.order[i]]
		if item.UserID == userID {
			out = append(out, *item)
		}
	}
	return out, nil
}

func (f *fakeStore) FindByID(id uuid.UUID) (*model.Notification, error) {
	notification, ok := f.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return notification, nil
}

func (f *fakeStore) MarkRead(notification *model.Notification) error {
	if notification.ReadAt == nil {
		now := time.Now()
		notification.ReadAt = &now
	}
	f.items[notification.ID] = notification
	return nil
}

func seed(t *testing.T, store *fakeStore, ownerID string, kind string) *model.Notification {
	t.Helper()

	notification := &model.Notification{
		UserID: uuid.MustParse(ownerID),
		Kind:   kind,
		Title:  "Judul",
		Body:   "Isi pesan",
	}
	if err := store.Create(notification); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return notification
}

func TestCreateStoresTheServiceNotification(t *testing.T) {
	store := newFakeStore()
	service := NewNotificationService(store)

	userID := uuid.NewString()
	notification, err := service.Create(&dto.CreateNotificationRequest{
		UserID: userID,
		Kind:   "booking.locked",
		Title:  "Kursi sedang dikunci",
		Body:   "Reservasi TICK-1 dibuat.",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if notification.UserID.String() != userID {
		t.Errorf("user = %s, want %s", notification.UserID, userID)
	}
	if notification.ID == uuid.Nil {
		t.Error("expected a generated id")
	}
	if len(store.items) != 1 {
		t.Errorf("stored = %d, want 1", len(store.items))
	}
}

func TestCreateRejectsMalformedUserID(t *testing.T) {
	service := NewNotificationService(newFakeStore())

	_, err := service.Create(&dto.CreateNotificationRequest{
		UserID: "nope",
		Kind:   "booking.locked",
		Title:  "Judul",
		Body:   "Isi",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

func TestListOnlyServesTheOwnersInbox(t *testing.T) {
	store := newFakeStore()
	service := NewNotificationService(store)

	ownerID := uuid.NewString()
	strangerID := uuid.NewString()
	seed(t, store, ownerID, "booking.locked")
	seed(t, store, strangerID, "payment.succeeded")

	t.Run("owner", func(t *testing.T) {
		got, err := service.List(&dto.ListNotificationsQuery{UserID: ownerID}, authz.Caller{UserID: ownerID})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Kind != "booking.locked" {
			t.Errorf("got %+v, want only the owner's notification", got)
		}
	})

	t.Run("stranger asking for someone else's inbox", func(t *testing.T) {
		if _, err := service.List(&dto.ListNotificationsQuery{UserID: ownerID}, authz.Caller{UserID: strangerID}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("no subject at all", func(t *testing.T) {
		if _, err := service.List(&dto.ListNotificationsQuery{UserID: ownerID}, authz.Caller{}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("internal service reads any inbox", func(t *testing.T) {
		got, err := service.List(&dto.ListNotificationsQuery{UserID: strangerID}, authz.Caller{Internal: true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Kind != "payment.succeeded" {
			t.Errorf("got %+v, want the inbox of strangerID", got)
		}
	})

	t.Run("malformed user id", func(t *testing.T) {
		if _, err := service.List(&dto.ListNotificationsQuery{UserID: "nope"}, authz.Caller{UserID: "nope"}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

func TestMarkReadRequiresOwnership(t *testing.T) {
	store := newFakeStore()
	service := NewNotificationService(store)

	ownerID := uuid.NewString()
	notification := seed(t, store, ownerID, "booking.locked")
	id := notification.ID.String()

	t.Run("owner", func(t *testing.T) {
		got, err := service.MarkRead(id, authz.Caller{UserID: ownerID})
		if err != nil {
			t.Fatalf("MarkRead: %v", err)
		}
		if got.ReadAt == nil {
			t.Fatal("expected ReadAt to be set")
		}

		// Idempotent: reading twice keeps the original timestamp.
		first := *got.ReadAt
		time.Sleep(2 * time.Millisecond)
		again, err := service.MarkRead(id, authz.Caller{UserID: ownerID})
		if err != nil {
			t.Fatalf("second MarkRead: %v", err)
		}
		if !again.ReadAt.Equal(first) {
			t.Error("ReadAt changed on a second read")
		}
	})

	t.Run("stranger", func(t *testing.T) {
		if _, err := service.MarkRead(id, authz.Caller{UserID: uuid.NewString()}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("internal service", func(t *testing.T) {
		other := seed(t, store, uuid.NewString(), "payment.succeeded")
		if _, err := service.MarkRead(other.ID.String(), authz.Caller{Internal: true}); err != nil {
			t.Errorf("MarkRead: %v", err)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		if _, err := service.MarkRead(uuid.NewString(), authz.Caller{UserID: ownerID}); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		if _, err := service.MarkRead("nope", authz.Caller{UserID: ownerID}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}
