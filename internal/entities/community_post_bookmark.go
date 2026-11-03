package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type CommunityPostBookmark struct {
	bun.BaseModel `bun:"table:community_post_bookmarks,alias:cpb"`

	PostID uuid.UUID `bun:"post_id,pk,type:uuid"`
	UserID uuid.UUID `bun:"user_id,pk,type:uuid"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}
