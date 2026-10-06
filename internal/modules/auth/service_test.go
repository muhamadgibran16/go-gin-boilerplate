package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"time"

	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ratelimit"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	testSecret          = "test-secret-with-at-least-32-characters"
	testMaxFailedLogins = 5
)

// fakeRepo is an in-memory UserRepository
type fakeRepo struct {
	users     map[uuid.UUID]*user.User
	createErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[uuid.UUID]*user.User{}}
}

func (r *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if u, ok := r.users[id]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			copied := *u
			return &copied, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) Create(_ context.Context, u *user.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	u.ID = uuid.New()
	copied := *u
	r.users[u.ID] = &copied
	return nil
}

func newTestService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	cfg := Config{JWTSecret: testSecret, AccessTokenTTL: time.Hour, RefreshTokenTTL: 7 * 24 * time.Hour}
	return NewService(repo, ratelimit.NewFailureCounter(testMaxFailedLogins, 15*time.Minute), cfg), repo
}

func TestRegisterNormalizesEmailAndRejectsDuplicates(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()

	u, err := svc.Register(ctx, " Bob ", "Bob@Example.COM", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.Email != "bob@example.com" || u.Name != "Bob" || u.Role != user.RoleUser {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.Password == "secret123" {
		t.Fatal("password must be hashed")
	}

	_, err = svc.Register(ctx, "Bob", "bob@example.com", "secret123")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("got %v, want ErrEmailTaken", err)
	}
}

func TestRegisterMapsDuplicateKeyToEmailTaken(t *testing.T) {
	svc, repo := newTestService()
	repo.createErr = gorm.ErrDuplicatedKey

	_, err := svc.Register(context.Background(), "Bob", "bob@example.com", "secret123")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("got %v, want ErrEmailTaken", err)
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	if _, err := svc.Register(ctx, "Bob", "bob@example.com", "secret123"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"success with different case", "BOB@example.com", "secret123", nil},
		{"wrong password", "bob@example.com", "wrong-password", ErrInvalidCredentials},
		{"unknown email", "alice@example.com", "secret123", ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tokens, err := svc.Login(ctx, tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if _, err := validateToken(tokens.AccessToken, testSecret, tokenTypeAccess); err != nil {
				t.Fatalf("access token invalid: %v", err)
			}
			if _, err := validateToken(tokens.RefreshToken, testSecret, tokenTypeRefresh); err != nil {
				t.Fatalf("refresh token invalid: %v", err)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestService()
	u, _ := svc.Register(ctx, "Bob", "bob@example.com", "secret123")
	_, tokens, err := svc.Login(ctx, "bob@example.com", "secret123")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("rejects access token", func(t *testing.T) {
		if _, err := svc.Refresh(ctx, tokens.AccessToken); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("got %v, want ErrInvalidRefreshToken", err)
		}
	})

	t.Run("uses current role from database", func(t *testing.T) {
		repo.users[u.ID].Role = user.RoleAdmin

		accessToken, err := svc.Refresh(ctx, tokens.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		c, err := validateToken(accessToken, testSecret, tokenTypeAccess)
		if err != nil {
			t.Fatal(err)
		}
		if c.Role != user.RoleAdmin {
			t.Fatalf("got role %q, want %q", c.Role, user.RoleAdmin)
		}
	})

	t.Run("rejects deleted user", func(t *testing.T) {
		delete(repo.users, u.ID)
		if _, err := svc.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("got %v, want ErrInvalidRefreshToken", err)
		}
	})
}

func TestCreateUserValidation(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()

	tests := []struct {
		name, userName, password string
		wantField                string
	}{
		{"blank name", "   ", "secret123", "name"},
		{"short password", "Bob", "12345", "password"},
		// 40 characters but 80 bytes: over the bcrypt limit, must be 400 and not 500
		{"multibyte password over 72 bytes", "Bob", strings.Repeat("é", 40), "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tt.userName, "bob@example.com", tt.password)
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation || appErr.Fields[tt.wantField] == "" {
				t.Fatalf("want validation error on %q, got %v", tt.wantField, err)
			}
		})
	}

	// 72 multibyte-free bytes is the maximum and must work
	if _, err := svc.Register(ctx, "Bob", "bob@example.com", strings.Repeat("a", 72)); err != nil {
		t.Fatalf("72-byte password must be accepted: %v", err)
	}
}

func TestCreateUserWithRole(t *testing.T) {
	svc, repo := newTestService()

	u, err := svc.CreateUser(context.Background(), "Admin", "Admin@Example.com", "secret123", user.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if repo.users[u.ID].Role != user.RoleAdmin || repo.users[u.ID].Email != "admin@example.com" {
		t.Fatalf("unexpected user: %+v", repo.users[u.ID])
	}
}

func TestLoginLockoutPerEmail(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	if _, err := svc.Register(ctx, "Bob", "bob@example.com", "secret123"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, "Alice", "alice@example.com", "secret123"); err != nil {
		t.Fatal(err)
	}

	isKind := func(err error, kind apperror.Kind) bool {
		var appErr *apperror.Error
		return errors.As(err, &appErr) && appErr.Kind == kind
	}

	// A successful login resets the counter, so earlier failures do not add up
	for i := 0; i < testMaxFailedLogins-1; i++ {
		_, _, _ = svc.Login(ctx, "bob@example.com", "wrong")
	}
	if _, _, err := svc.Login(ctx, "bob@example.com", "secret123"); err != nil {
		t.Fatalf("login before the limit must work: %v", err)
	}

	for i := 0; i < testMaxFailedLogins; i++ {
		if _, _, err := svc.Login(ctx, "BOB@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("failure %d: got %v, want ErrInvalidCredentials", i+1, err)
		}
	}

	// Locked: even the right password is refused, whatever the email casing
	if _, _, err := svc.Login(ctx, "bob@example.com", "secret123"); !isKind(err, apperror.KindTooManyRequests) {
		t.Fatalf("locked account: got %v, want too many requests", err)
	}

	// Other accounts are not affected
	if _, _, err := svc.Login(ctx, "alice@example.com", "secret123"); err != nil {
		t.Fatalf("other account must not be locked: %v", err)
	}

	// Unknown emails are locked the same way, so the response does not reveal which accounts exist
	for i := 0; i < testMaxFailedLogins; i++ {
		_, _, _ = svc.Login(ctx, "ghost@example.com", "wrong")
	}
	if _, _, err := svc.Login(ctx, "ghost@example.com", "wrong"); !isKind(err, apperror.KindTooManyRequests) {
		t.Fatalf("unknown email: got %v, want too many requests", err)
	}
}
