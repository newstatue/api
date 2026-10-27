package database

import (
	"context"

	"github.com/newstatue/api/internal/entities"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

func Migrate(ctx context.Context, db *bun.DB) error {
	migrator, err := migrate.NewAutoMigrator(
		db,
		migrate.WithModel((*entities.User)(nil)),
		migrate.WithMigrationsDirectoryAuto("migrations"),
	)
	if err != nil {
		return err
	}

	_, err = migrator.Migrate(ctx)
	return err
}
