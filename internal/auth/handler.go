package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/newstatue/api/internal/entities"
	"github.com/newstatue/api/internal/session"
	"github.com/newstatue/api/internal/util"
	"github.com/uptrace/bun"
)

type Handler struct {
	DB      *bun.DB
	Session *session.Manager
}

type RegisterUserInput struct {
	Body RegisterUser
}

type RegisterUserOutput struct {
	Body User
}

func (h *Handler) RegisterUser(ctx context.Context, i *RegisterUserInput) (*RegisterUserOutput, error) {
	// 1. 检查邮箱是否存在
	email := util.NormalizeEmail(i.Body.Email)
	exists, err := h.DB.NewSelect().Model((*entities.User)(nil)).Where("email = ?", email).Exists(ctx)
	if err != nil {
		log.Println(err)
		return nil, huma.Error500InternalServerError("数据库查询失败")
	}
	if exists {
		return nil, huma.Error409Conflict("邮箱已被注册")
	}

	// 2. 使用 Argon2id 哈希密码
	password, err := util.HashPassword(i.Body.Password)
	if err != nil {
		log.Println(err)
		return nil, huma.Error500InternalServerError("哈希密码失败")
	}

	// 3. 使用 Bun ORM 创建用户
	user := &entities.User{
		Name:     i.Body.Name,
		Email:    email,
		Password: &password,
		Image:    i.Body.Image,
	}

	_, err = h.DB.NewInsert().Model(user).Returning("id").Exec(ctx)
	if err != nil {
		log.Println(err)
		return nil, huma.Error500InternalServerError("创建用户失败")
	}

	// 4. 返回用户信息
	return &RegisterUserOutput{
		Body: User{
			ID:            user.ID,
			Name:          user.Name,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			Image:         user.Image,
		},
	}, nil
}

type LoginUserInput struct {
	SessionID string `cookie:"session"`
	UserAgent string `header:"User-Agent"`
	Body      LoginUser
}

type LoginUserOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      User
}

func (h *Handler) LoginUser(ctx context.Context, i *LoginUserInput) (*LoginUserOutput, error) {
	email := util.NormalizeEmail(i.Body.Email)

	// 1. 查询用户
	user := new(entities.User)
	err := h.DB.NewSelect().Model(user).Where("email = ?", email).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error401Unauthorized("邮箱或密码错误")
	}
	if err != nil {
		log.Println(err)
		return nil, huma.Error500InternalServerError("数据库查询失败")
	}

	// 2. 验证密码
	if user.Password == nil {
		return nil, huma.Error401Unauthorized("邮箱或密码错误")
	}
	valid, err := util.VerifyPassword(
		i.Body.Password,
		*user.Password,
	)
	if err != nil {
		log.Println(err)
		return nil, huma.Error500InternalServerError("密码验证失败")
	}
	if !valid {
		return nil, huma.Error401Unauthorized("邮箱或密码错误")
	}

	// 3. 验证成功，创建session
	sessionID, err := h.Session.Create(ctx, session.Session{
		UserID:    user.ID.String(),
		UserAgent: i.UserAgent,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("处理会话失败")
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(session.TTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	// 4. 返回用户
	return &LoginUserOutput{
		SetCookie: cookie.String(),
		Body: User{
			ID:            user.ID,
			Name:          user.Name,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			Image:         user.Image,
		},
	}, nil
}
