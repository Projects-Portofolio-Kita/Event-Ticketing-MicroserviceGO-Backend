package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/dto"
	authjwt "github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/jwt"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/auth-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrGoogleNotConfigured = errors.New("google login is not configured")
	ErrGoogleTokenInvalid  = errors.New("google token is invalid")
)

// userStore is the consumer-side slice of repository.UserRepository that the
// service actually uses. Depending on this narrow interface keeps the service
// testable with an in-memory fake; *repository.UserRepository satisfies it.
type userStore interface {
	CreateUser(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
	FindByGoogleSub(sub string) (*model.User, error)
	SaveRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindValidRefreshToken(tokenHash string) (*model.RefreshToken, error)
	RevokeRefreshToken(id uuid.UUID) error
}

type AuthService struct {
	repo             userStore
	jwt              *authjwt.Manager
	googleClientID   string
	googleHTTPClient *http.Client
}

func NewAuthService(repo userStore, jwt *authjwt.Manager, googleClientID string) *AuthService {
	return &AuthService{
		repo:             repo,
		jwt:              jwt,
		googleClientID:   strings.TrimSpace(googleClientID),
		googleHTTPClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// issueSession creates an access JWT + opaque refresh token (persisted hashed).
func (s *AuthService) issueSession(user *model.User) (dto.SessionResponse, error) {
	access, err := s.jwt.GenerateAccess(user.ID, user.Email)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	refresh, err := s.jwt.GenerateRefresh()
	if err != nil {
		return dto.SessionResponse{}, err
	}
	if err := s.repo.SaveRefreshToken(user.ID, authjwt.HashRefresh(refresh), time.Now().Add(s.jwt.RefreshTTL())); err != nil {
		return dto.SessionResponse{}, err
	}
	return dto.SessionResponse{
		AccessToken:  access,
		Token:        access,
		RefreshToken: refresh,
		User:         user.Public(),
	}, nil
}

func (s *AuthService) Register(name, email, password string) (dto.SessionResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dto.SessionResponse{}, err
	}
	user := &model.User{
		Name:         strings.TrimSpace(name),
		Email:        normalizeEmail(email),
		PasswordHash: string(hash),
		Provider:     model.ProviderLocal,
	}
	if err := s.repo.CreateUser(user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return dto.SessionResponse{}, repository.ErrDuplicateEmail
		}
		return dto.SessionResponse{}, err
	}
	return s.issueSession(user)
}

func (s *AuthService) Login(email, password string) (dto.SessionResponse, error) {
	user, err := s.repo.FindByEmail(normalizeEmail(email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.SessionResponse{}, ErrInvalidCredentials
		}
		return dto.SessionResponse{}, err
	}
	if user.Provider != model.ProviderLocal || user.PasswordHash == "" {
		return dto.SessionResponse{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return dto.SessionResponse{}, ErrInvalidCredentials
	}
	return s.issueSession(user)
}

func (s *AuthService) Refresh(rawRefresh string) (dto.SessionResponse, error) {
	stored, err := s.repo.FindValidRefreshToken(authjwt.HashRefresh(strings.TrimSpace(rawRefresh)))
	if err != nil {
		return dto.SessionResponse{}, ErrInvalidCredentials
	}
	user, err := s.repo.FindByID(stored.UserID)
	if err != nil {
		return dto.SessionResponse{}, ErrInvalidCredentials
	}
	// Rotate: revoke the used token, then issue a fresh pair.
	if err := s.repo.RevokeRefreshToken(stored.ID); err != nil {
		return dto.SessionResponse{}, err
	}
	return s.issueSession(user)
}

// googleTokenInfo is the subset of the tokeninfo response we verify.
type googleTokenInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
}

// LoginWithGoogle verifies the GIS ID token via Google's tokeninfo endpoint
// (signature + expiry checked by Google), then enforces our audience and
// verified-email requirements before issuing our own session.
func (s *AuthService) LoginWithGoogle(idToken string) (dto.SessionResponse, error) {
	if s.googleClientID == "" {
		return dto.SessionResponse{}, ErrGoogleNotConfigured
	}
	endpoint := "https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=" + url.QueryEscape(strings.TrimSpace(idToken))
	resp, err := s.googleHTTPClient.Get(endpoint)
	if err != nil {
		return dto.SessionResponse{}, ErrGoogleTokenInvalid
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return dto.SessionResponse{}, ErrGoogleTokenInvalid
	}
	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return dto.SessionResponse{}, ErrGoogleTokenInvalid
	}
	if info.Sub == "" || info.Aud != s.googleClientID || info.Email == "" || info.EmailVerified != "true" {
		return dto.SessionResponse{}, ErrGoogleTokenInvalid
	}

	// Existing Google-linked account?
	if user, err := s.repo.FindByGoogleSub(info.Sub); err == nil {
		return s.issueSession(user)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.SessionResponse{}, err
	}
	// Same email registered via local form? Link it instead of duplicating.
	if user, err := s.repo.FindByEmail(normalizeEmail(info.Email)); err == nil {
		return s.issueSession(user)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.SessionResponse{}, err
	}

	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = strings.Split(info.Email, "@")[0]
	}
	user := &model.User{
		Name:      name,
		Email:     normalizeEmail(info.Email),
		AvatarURL: info.Picture,
		Provider:  model.ProviderGoogle,
		GoogleSub: &info.Sub,
	}
	if err := s.repo.CreateUser(user); err != nil {
		// Race: account appeared between check and insert — retry lookup.
		if errors.Is(err, repository.ErrDuplicateEmail) {
			if existing, lerr := s.repo.FindByEmail(user.Email); lerr == nil {
				return s.issueSession(existing)
			}
		}
		return dto.SessionResponse{}, fmt.Errorf("failed to create google user: %w", err)
	}
	return s.issueSession(user)
}

func (s *AuthService) Me(userID uuid.UUID) (*model.User, error) {
	return s.repo.FindByID(userID)
}
