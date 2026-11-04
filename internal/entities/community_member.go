package entities

import (
	"time"
	"uuid"

	"github.com/uptrace/bun"
)

type CommunityRole string

const (
	CommunityRoleOwner     CommunityRole = "owner"
	CommunityRoleModerator CommunityRole = "moderator"
	CommunityRoleMember    CommunityRole = "member"
)

type CommunityMember struct {
	bun.BaseModel `bun:"table:community_members,alias:cm"`

	CommunityID uuid.UUID     `bun:"community_id,pk,type:uuid"`
	UserID      uuid.UUID     `bun:"user_id,pk,type:uuid"`
	Role        CommunityRole `bun:"role,notnull,default:'member'"`

	Community *Community `bun:"rel:belongs-to,join:community_id=id"`
	User      *User      `bun:"rel:belongs-to,join:user_id=id"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}
