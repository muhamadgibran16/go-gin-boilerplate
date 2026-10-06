package user

import (
	"context"
	"errors"
	"testing"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeRepo is an in-memory UserRepository
type fakeRepo struct {
	users   map[uuid.UUID]*User
	findErr error
}

func (r *fakeRepo) FindAll(_ context.Context, _ ListQuery) ([]User, int64, error) {
	var users []User
	for _, u := range r.users {
		users = append(users, *u)
	}
	return users, int64(len(users)), nil
}

func (r *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if u, ok := r.users[id]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeRepo) Update(_ context.Context, user *User) error {
	copied := *user
	r.users[user.ID] = &copied
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.users, id)
	return nil
}

func setup() (*Service, *fakeRepo, *User, *User) {
	admin := &User{ID: uuid.New(), Name: "Admin", Role: RoleAdmin}
	user := &User{ID: uuid.New(), Name: "User", Role: RoleUser}
	repo := &fakeRepo{users: map[uuid.UUID]*User{admin.ID: admin, user.ID: user}}
	return NewService(repo), repo, admin, user
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _ := setup()

	if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}

	// Database errors must not be reported as "not found"
	dbErr := errors.New("connection refused")
	repo.findErr = dbErr
	if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, dbErr) {
		t.Fatalf("got %v, want the database error", err)
	}
}

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("admin updates another user", func(t *testing.T) {
		svc, repo, admin, user := setup()
		updated, err := svc.Update(ctx, admin.ID, user.ID, UpdateInput{Name: "Renamed", Role: RoleAdmin})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Name != "Renamed" || repo.users[user.ID].Role != RoleAdmin {
			t.Fatalf("user not updated: %+v", repo.users[user.ID])
		}
	})

	t.Run("admin cannot change own role", func(t *testing.T) {
		svc, repo, admin, _ := setup()
		_, err := svc.Update(ctx, admin.ID, admin.ID, UpdateInput{Role: RoleUser})
		if !errors.Is(err, ErrSelfModification) {
			t.Fatalf("got %v, want ErrSelfModification", err)
		}
		if repo.users[admin.ID].Role != RoleAdmin {
			t.Fatal("role must not change")
		}
	})

	t.Run("admin can change own name", func(t *testing.T) {
		svc, _, admin, _ := setup()
		if _, err := svc.Update(ctx, admin.ID, admin.ID, UpdateInput{Name: "New Name", Role: RoleAdmin}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("name is trimmed and checked after trimming", func(t *testing.T) {
		svc, repo, admin, user := setup()
		for _, name := range []string{"   ", " ab "} {
			_, err := svc.Update(ctx, admin.ID, user.ID, UpdateInput{Name: name})
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.Fields["name"] == "" {
				t.Fatalf("name %q: want validation error, got %v", name, err)
			}
		}
		if _, err := svc.Update(ctx, admin.ID, user.ID, UpdateInput{Name: "  Budi  "}); err != nil {
			t.Fatal(err)
		}
		if got := repo.users[user.ID].Name; got != "Budi" {
			t.Fatalf("name = %q, want trimmed %q", got, "Budi")
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _, admin, _ := setup()
		if _, err := svc.Update(ctx, admin.ID, uuid.New(), UpdateInput{Name: "X"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
	})
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	svc, repo, admin, user := setup()

	if err := svc.Delete(ctx, admin.ID, admin.ID); !errors.Is(err, ErrSelfModification) {
		t.Fatalf("got %v, want ErrSelfModification", err)
	}
	if err := svc.Delete(ctx, admin.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, admin.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.users[user.ID]; ok {
		t.Fatal("user must be deleted")
	}
}
