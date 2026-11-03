package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type CommunityPostVote struct {
	bun.BaseModel `bun:"table:community_post_votes,alias:cpv"`

	PostID uuid.UUID `bun:"post_id,pk,type:uuid"`
	UserID uuid.UUID `bun:"user_id,pk,type:uuid"`

	Value int16 `bun:"value,notnull"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}
