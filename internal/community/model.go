package community

import (
	"time"
	"uuid"
)

type Community struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateCommunity struct {
	Slug        string    `json:"slug" minLength:"3" maxLength:"32" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$"`
	Name        string    `json:"name" minLength:"1" maxLength:"100"`
	Description string    `json:"description" maxLength:"1000"`
	OwnerID     uuid.UUID `json:"ownerId"`
}

type UpdateCommunity struct {
	Name        *string `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Description *string `json:"description,omitempty" maxLength:"1000"`
}

type JoinCommunity struct {
	Joined bool `json:"joined"`
}
