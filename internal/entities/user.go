package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID            uuid.UUID `bun:"id,pk,type:uuid,default:uuidv7()"`
	Name          string    `bun:"name,notnull"`
	Email         string    `bun:"email,notnull,unique"`
	EmailVerified bool      `bun:"email_verified,notnull,default:false"`
	Password      *string   `bun:"password"`
	Image         *string   `bun:"image"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
