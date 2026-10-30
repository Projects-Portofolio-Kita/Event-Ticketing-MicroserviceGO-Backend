package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/dto"
	authjwt "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// fakeUserStore is an in-memory userStore; it mirrors the repository's
// error contract (ErrDuplicateEmail, gorm.ErrRecordNotFound, ...).
type fakeUserStore struct {
	users         []*model.User
	refreshTokens []*model.RefreshToken
}

func (f *fakeUserStore) CreateUser(user *model.User) error {
	for _, existing := range f.users {
		if existing.Email == user.Email {
			return repository.ErrDuplicateEmail
		}
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	stored := *user
	f.users = append(f.users, &stored)
	return nil
}

func (f *fakeUserStore) FindByEmail(email string) (*model.User, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	for _, user := range f.users {
		if user.Email == key {
			stored := *user
			return &stored, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserStore) FindByID(id uuid.UUID) (*model.User, error) {
	for _, user := range f.users {
		if user.ID == id {
			stored := *user
			return &stored, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserStore) FindByGoogleSub(sub string) (*model.User, error) {
	for _, user := range f.users {
		if user.GoogleSub != nil && *user.GoogleSub == sub {
			stored := *user
			return &stored, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserStore) SaveRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	f.refreshTokens = append(f.refreshTokens, &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	return nil
}

func (f *fakeUserStore) FindValidRefreshToken(tokenHash string) (*model.RefreshToken, error) {
	for _, token := range f.refreshTokens {
		if token.TokenHash == tokenHash && !token.Revoked && token.ExpiresAt.After(time.Now()) {
			stored := *token
			return &stored, nil
		}
	}
	return nil, repository.ErrRefreshNotFound
}

func (f *fakeUserStore) RevokeRefreshToken(id uuid.UUID) error {
	for _, token := range f.refreshTokens {
		if token.ID == id {
			token.Revoked = true
		}
	}
	return nil
}

func (f *fakeUserStore) UpdateUser(user *model.User) error {
	for i, existing := range f.users {
		if existing.ID == user.ID {
			stored := *user
			stored.UpdatedAt = time.Now()
			f.users[i] = &stored
			*user = stored
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// newTestService builds a service over the fake store. The Google client ID is
// left empty so no code path can reach the network.
func newTestService() (*AuthService, *fakeUserStore, *authjwt.Manager) {
	store := &fakeUserStore{}
	manager := authjwt.NewManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	return NewAuthService(store, manager, ""), store, manager
}

func mustRegister(t *testing.T, svc *AuthService, name, email, password string) dto.SessionResponse {
	t.Helper()
	session, err := svc.Register(name, email, password, false)
	if err != nil {
		t.Fatalf("Register(%q): %v", email, err)
	}
	return session
}

func TestRegisterIssuesParseableSession(t *testing.T) {
	svc, store, manager := newTestService()

	session := mustRegister(t, svc, "Alice", "  Alice@Example.COM ", "secret1")

	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatalf("session = %+v, want non-empty access and refresh tokens", session)
	}
	if session.AccessToken != session.Token {
		t.Errorf("Token %q differs from AccessToken %q", session.Token, session.AccessToken)
	}

	userID, email, err := manager.ParseAccess(session.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if len(store.users) != 1 {
		t.Fatalf("stored users = %d, want 1", len(store.users))
	}
	if userID != store.users[0].ID.String() {
		t.Errorf("sub = %q, want %q", userID, store.users[0].ID)
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want normalized alice@example.com", email)
	}
	if session.User.Email != "alice@example.com" {
		t.Errorf("session user email = %q, want alice@example.com", session.User.Email)
	}
	if len(store.refreshTokens) != 1 {
		t.Fatalf("stored refresh tokens = %d, want 1", len(store.refreshTokens))
	}
	if store.refreshTokens[0].TokenHash == session.RefreshToken {
		t.Error("raw refresh token was persisted; only its hash may be stored")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.users[0].PasswordHash), []byte("secret1")); err != nil {
		t.Errorf("stored password hash does not verify: %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")

	// Case and whitespace must not bypass the unique check.
	_, err := svc.Register("Alice 2", "  ALICE@example.com ", "secret2", false)
	if !errors.Is(err, repository.ErrDuplicateEmail) {
		t.Fatalf("Register duplicate = %v, want ErrDuplicateEmail", err)
	}
}

func TestRegisterPersistsCanPostEventFlag(t *testing.T) {
	svc, store, _ := newTestService()

	session, err := svc.Register("Poster", "poster@example.com", "secret1", true)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if !session.User.CanPostEvent {
		t.Fatal("session user canPostEvent = false, want true")
	}
	if len(store.users) != 1 {
		t.Fatalf("stored users = %d, want 1", len(store.users))
	}
	if !store.users[0].CanPostEvent {
		t.Fatal("stored user canPostEvent = false, want true")
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	t.Run("unknown user", func(t *testing.T) {
		svc, _, _ := newTestService()
		if _, err := svc.Login("ghost@example.com", "secret1"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, _, _ := newTestService()
		mustRegister(t, svc, "Alice", "alice@example.com", "secret1")
		if _, err := svc.Login("alice@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("google-backed account has no local password", func(t *testing.T) {
		svc, store, _ := newTestService()
		sub := "google-sub-1"
		store.users = append(store.users, &model.User{
			ID:        uuid.New(),
			Name:      "Bob",
			Email:     "bob@example.com",
			Provider:  model.ProviderGoogle,
			GoogleSub: &sub,
		})
		if _, err := svc.Login("bob@example.com", "anything"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestLoginSuccessReturnsParseableSession(t *testing.T) {
	svc, _, manager := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")

	session, err := svc.Login("ALICE@example.com", "secret1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	userID, email, err := manager.ParseAccess(session.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", email)
	}
	if userID == "" {
		t.Error("sub is empty, want the user ID")
	}
	if session.RefreshToken == "" {
		t.Error("refresh token is empty")
	}
	if session.User.Name != "Alice" {
		t.Errorf("user name = %q, want Alice", session.User.Name)
	}
}

func TestRefreshRotatesTokenPair(t *testing.T) {
	svc, store, manager := newTestService()
	first := mustRegister(t, svc, "Alice", "alice@example.com", "secret1")

	second, err := svc.Refresh(first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}
	if _, _, err := manager.ParseAccess(second.AccessToken); err != nil {
		t.Errorf("rotated access token does not parse: %v", err)
	}
	if len(store.refreshTokens) != 2 {
		t.Fatalf("stored refresh tokens = %d, want 2", len(store.refreshTokens))
	}

	// The consumed token must be revoked: a second refresh fails.
	if _, err := svc.Refresh(first.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("re-using a rotated token = %v, want ErrInvalidCredentials", err)
	}
}

func TestRefreshRejectsBadTokens(t *testing.T) {
	t.Run("unknown token", func(t *testing.T) {
		svc, _, _ := newTestService()
		if _, err := svc.Refresh("deadbeefdeadbeef"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Refresh = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		svc, store, _ := newTestService()
		mustRegister(t, svc, "Alice", "alice@example.com", "secret1")
		raw := "expired-refresh-token-value"
		store.refreshTokens = append(store.refreshTokens, &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    store.users[0].ID,
			TokenHash: authjwt.HashRefresh(raw),
			ExpiresAt: time.Now().Add(-time.Minute),
		})
		if _, err := svc.Refresh(raw); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Refresh = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestMeReturnsUser(t *testing.T) {
	svc, store, _ := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")

	user, err := svc.Me(store.users[0].ID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}

	if _, err := svc.Me(uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Me(unknown) = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestLoginWithGoogleWithoutClientID(t *testing.T) {
	svc, _, _ := newTestService()

	if _, err := svc.LoginWithGoogle("some-google-id-token"); !errors.Is(err, ErrGoogleNotConfigured) {
		t.Fatalf("LoginWithGoogle = %v, want ErrGoogleNotConfigured", err)
	}
}

// --- profile (PATCH /me) ----------------------------------------------------

func strptr(s string) *string { return &s }

func TestUpdateProfileChangesNameAndAvatar(t *testing.T) {
	svc, store, _ := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")

	user, err := svc.UpdateProfile(store.users[0].ID, "  Alice Tan  ", strptr("https://cdn.example.com/a.png"))
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	if user.Name != "Alice Tan" {
		t.Errorf("name = %q, want the trimmed Alice Tan", user.Name)
	}
	if user.AvatarURL != "https://cdn.example.com/a.png" {
		t.Errorf("avatar = %q, want the new URL", user.AvatarURL)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email changed to %q — profile edits must not touch it", user.Email)
	}
	// The change is persisted for the next /me read.
	fresh, err := svc.Me(store.users[0].ID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if fresh.Name != "Alice Tan" {
		t.Errorf("stored name = %q, want Alice Tan", fresh.Name)
	}
}

func TestUpdateProfileAvatarSemantics(t *testing.T) {
	svc, store, _ := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")
	userID := store.users[0].ID

	// Seed an avatar first.
	if _, err := svc.UpdateProfile(userID, "Alice", strptr("https://cdn.example.com/a.png")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("omitted avatar is left untouched", func(t *testing.T) {
		user, err := svc.UpdateProfile(userID, "Alice", nil)
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if user.AvatarURL != "https://cdn.example.com/a.png" {
			t.Errorf("avatar = %q, want it unchanged", user.AvatarURL)
		}
	})

	t.Run("empty avatar clears it", func(t *testing.T) {
		user, err := svc.UpdateProfile(userID, "Alice", strptr("   "))
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if user.AvatarURL != "" {
			t.Errorf("avatar = %q, want it cleared for the initials fallback", user.AvatarURL)
		}
	})
}

func TestUpdateProfileRejectsBadInput(t *testing.T) {
	svc, store, _ := newTestService()
	mustRegister(t, svc, "Alice", "alice@example.com", "secret1")
	userID := store.users[0].ID

	for name, tc := range map[string]struct {
		name   string
		avatar *string
	}{
		"blank name":       {name: "   ", avatar: nil},
		"oversized name":   {name: strings.Repeat("a", 101), avatar: nil},
		"avatar not http":  {name: "Alice", avatar: strptr("ftp://cdn.example.com/a.png")},
		"oversized avatar": {name: "Alice", avatar: strptr("https://cdn.example.com/" + strings.Repeat("a", 600))},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.UpdateProfile(userID, tc.name, tc.avatar); !errors.Is(err, ErrInvalidProfile) {
				t.Errorf("err = %v, want ErrInvalidProfile", err)
			}
			// The stored profile must be untouched.
			user, err := svc.Me(userID)
			if err != nil {
				t.Fatalf("Me: %v", err)
			}
			if user.Name != "Alice" {
				t.Errorf("name = %q, want the original Alice", user.Name)
			}
		})
	}
}

func TestUpdateProfileUnknownUser(t *testing.T) {
	svc, _, _ := newTestService()

	if _, err := svc.UpdateProfile(uuid.New(), "Ghost", nil); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}
