package auth

import "uuid"

type User struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Image         *string   `json:"image"`
}

type RegisterUser struct {
	Name     string  `json:"name" required:"true" minLength:"2" maxLength:"50"`
	Email    string  `json:"email" required:"true" format:"email"`
	Image    *string `json:"image,omitempty" format:"uri"`
	Password string  `json:"password" required:"true" minLength:"8" maxLength:"128"`
}

type LoginUser struct {
	Email    string `json:"email" format:"email" required:"true"`
	Password string `json:"password" required:"true"`
}
