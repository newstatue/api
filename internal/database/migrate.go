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
		migrate.WithModel((*entities.Community)(nil)),
		migrate.WithModel((*entities.CommunityMember)(nil)),
		migrate.WithModel((*entities.CommunityPost)(nil)),
		migrate.WithModel((*entities.CommunityPostVote)(nil)),
		migrate.WithModel((*entities.CommunityPostBookmark)(nil)),
		migrate.WithMigrationsDirectoryAuto("migrations"),
	)
	if err != nil {
		return err
	}

	_, err = migrator.Migrate(ctx)
	return err
}
