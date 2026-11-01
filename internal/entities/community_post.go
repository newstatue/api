package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type CommunityPost struct {
	bun.BaseModel `bun:"table:community_posts,alias:cp"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuidv7()"`
	CommunityID uuid.UUID `bun:"community_id,notnull,type:uuid"`
	AuthorID    uuid.UUID `bun:"author_id,notnull,type:uuid"`

	Title   string `bun:"title,notnull"`
	Content string `bun:"content,notnull,type:text"`

	Community *Community `bun:"rel:belongs-to,join:community_id=id"`
	Author    *User      `bun:"rel:belongs-to,join:author_id=id"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
