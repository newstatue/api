package user

import "github.com/danielgtaylor/huma/v2"

type PutAvatarForm struct {
	Image huma.FormFile `form:"image" contentType:"image/jpeg,image/png" required:"true"`
}
