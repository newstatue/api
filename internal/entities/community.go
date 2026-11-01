package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type Community struct {
	bun.BaseModel `bun:"table:communities,alias:c"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuidv7()"`
	Slug        string    `bun:"slug,notnull,unique"`
	Name        string    `bun:"name,notnull"`
	Description string    `bun:"description,notnull,default:''"`

	OwnerID uuid.UUID `bun:"owner_id,notnull,type:uuid"`
	Owner   *User     `bun:"rel:belongs-to,join:owner_id=id"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
