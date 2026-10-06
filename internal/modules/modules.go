// Package modules groups the application's feature modules (auth, user, health, ...).
// Each module owns its handler, service, repository, model and DTOs.
package modules

import "github.com/gibran/go-gin-boilerplate/internal/modules/user"

// Models returns every GORM model in the application.
// It is used by `make db:push` to sync the schema in development, so register new models here.
func Models() []any {
	return []any{
		&user.User{},
	}
}
