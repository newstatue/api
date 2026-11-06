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

	IsMember   bool   `json:"isMember"`
	MemberRole string `json:"memberRole"`
}

type Create struct {
	Slug        string    `json:"slug" minLength:"3" maxLength:"32" pattern:"^[a-z0-9]+(-[a-z0-9]+)*$"`
	Name        string    `json:"name" minLength:"1" maxLength:"100"`
	Description string    `json:"description" maxLength:"1000"`
	OwnerID     uuid.UUID `json:"ownerId"`
}

type Update struct {
	Name        *string `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Description *string `json:"description,omitempty" maxLength:"1000"`
}

type Post struct {
	ID          uuid.UUID `json:"id"`
	CommunityID uuid.UUID `json:"communityId"`
	AuthorID    uuid.UUID `json:"authorId"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	VoteCount   int64     `json:"voteCount"`
}

type CreatePost struct {
	Title   string `json:"title" minLength:"1" maxLength:"200"`
	Content string `json:"content" minLength:"1" maxLength:"50000"`
}

type UpdatePost struct {
	Title   *string `json:"title,omitempty" minLength:"1" maxLength:"200"`
	Content *string `json:"content,omitempty" minLength:"1"`
}

type PutMembership struct {
	Joined bool `json:"joined"`
}

type PutPostVote struct {
	Value int16 `json:"value" enum:"-1,0,1"`
}

type PutPostBookmark struct {
	Bookmarked bool `json:"bookmarked"`
}
