package auth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ctxlog"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken          = apperror.Conflict("email already registered")
	ErrInvalidCredentials  = apperror.Unauthorized("invalid email or password")
	ErrInvalidRefreshToken = apperror.Unauthorized("invalid or expired refresh token")
)

const (
	minPasswordLength = 6
	// maxPasswordBytes is the bcrypt input limit
	maxPasswordBytes = 72
)

// dummyHash is compared against when the email is unknown, so login takes
// the same time whether or not the account exists (prevents user enumeration).
var dummyHash, _ = hashPassword("dummy-password-for-timing")

// Config holds the token settings of the auth module
type Config struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// userRepository is the user data access the auth service needs.
// *user.Repository satisfies it; tests use a fake.
type userRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	Create(ctx context.Context, u *user.User) error
}

// Tokens is a freshly issued access/refresh token pair
type Tokens struct {
	AccessToken  string
	RefreshToken string
}

// Service handles authentication logic
type Service struct {
	users         userRepository
	loginFailures failureCounter
	config        Config
}

// failureCounter tracks failed logins per email. *ratelimit.FailureCounter satisfies it.
type failureCounter interface {
	Blocked(ctx context.Context, key string) (time.Duration, bool, error)
	Fail(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}

// NewService creates a new Service. loginFailures limits failed logins per email,
// which stops brute force against one account even when it comes from many IP addresses.
func NewService(users userRepository, loginFailures failureCounter, cfg Config) *Service {
	return &Service{users: users, loginFailures: loginFailures, config: cfg}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Register creates a new user with the default role
func (s *Service) Register(ctx context.Context, name, email, password string) (*user.User, error) {
	return s.CreateUser(ctx, name, email, password, user.RoleUser)
}

// CreateUser creates a user with the given role. Register uses it for self sign-up;
// the `make db:create-admin` command uses it to create the first admin.
func (s *Service) CreateUser(ctx context.Context, name, email, password, role string) (*user.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)

	if err := validateCredentials(name, email, password); err != nil {
		return nil, err
	}

	existing, err := s.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		Name:     name,
		Email:    email,
		Password: hashed,
		Role:     role,
	}

	if err := s.users.Create(ctx, u); err != nil {
		// Concurrent registration with the same email hits the unique index
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailTaken.Wrap(err)
		}
		return nil, err
	}

	ctxlog.From(ctx).Info("user created", zap.String("created_user_id", u.ID.String()), zap.String("role", role))
	return u, nil
}

// validateCredentials checks rules the request binding cannot express. It also protects
// callers that do not go through HTTP binding, such as the create-admin command.
func validateCredentials(name, email, password string) error {
	fields := map[string]string{}
	if name == "" {
		fields["name"] = "name is required"
	}
	if email == "" {
		fields["email"] = "email is required"
	}
	if utf8.RuneCountInString(password) < minPasswordLength {
		fields["password"] = fmt.Sprintf("password must be at least %d characters long", minPasswordLength)
	} else if len(password) > maxPasswordBytes {
		// bcrypt only accepts 72 bytes; non-ASCII characters take 2-4 bytes each
		fields["password"] = fmt.Sprintf("password must be at most %d bytes (non-ASCII characters count as several bytes)", maxPasswordBytes)
	}
	if len(fields) > 0 {
		return apperror.Validation(fields)
	}
	return nil
}

// Login validates credentials and issues an access and a refresh token
//
// After too many failed attempts on an email, every login to it is refused for a while,
// even with the right password. Unknown emails are counted too, so the response does not
// reveal whether an account exists.
func (s *Service) Login(ctx context.Context, email, password string) (*user.User, *Tokens, error) {
	email = normalizeEmail(email)

	retryAfter, blocked, err := s.loginFailures.Blocked(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if blocked {
		minutes := int(math.Ceil(retryAfter.Minutes()))
		return nil, nil, apperror.TooManyRequests(
			fmt.Sprintf("too many failed login attempts, try again in %d minute(s)", max(minutes, 1)))
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			comparePassword(dummyHash, password)
			return nil, nil, s.loginFailed(ctx, email)
		}
		return nil, nil, err
	}

	if !comparePassword(u.Password, password) {
		return nil, nil, s.loginFailed(ctx, email)
	}

	if err := s.loginFailures.Reset(ctx, email); err != nil {
		return nil, nil, err
	}

	accessToken, err := s.generateAccessToken(u)
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := generateToken(u.ID, u.Role, tokenTypeRefresh, s.config.JWTSecret, s.config.RefreshTokenTTL)
	if err != nil {
		return nil, nil, err
	}

	return u, &Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Refresh validates a refresh token and issues a new access token.
// The user is reloaded from the database so deleted users cannot refresh and
// role changes are reflected in the new access token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, error) {
	c, err := validateToken(refreshToken, s.config.JWTSecret, tokenTypeRefresh)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	u, err := s.users.FindByID(ctx, c.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrInvalidRefreshToken
		}
		return "", err
	}

	return s.generateAccessToken(u)
}

// loginFailed records a failed attempt and returns the error for the client
func (s *Service) loginFailed(ctx context.Context, email string) error {
	if err := s.loginFailures.Fail(ctx, email); err != nil {
		return err
	}
	return ErrInvalidCredentials
}

func (s *Service) generateAccessToken(u *user.User) (string, error) {
	return generateToken(u.ID, u.Role, tokenTypeAccess, s.config.JWTSecret, s.config.AccessTokenTTL)
}
