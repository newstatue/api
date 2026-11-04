package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"
	"github.com/newstatue/api/internal/entities"
	"github.com/newstatue/api/internal/image"
	"github.com/newstatue/api/internal/session"
	"github.com/uptrace/bun"
	"gocloud.dev/blob"
)

type Handler struct {
	DB      *bun.DB
	Session *session.Manager
	Bucket  *blob.Bucket
}

type PutAvatarInput struct {
	RawBody huma.MultipartFormFiles[PutAvatarForm]
}

func (h *Handler) PutAvatar(ctx context.Context, i *PutAvatarInput) (*struct{}, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

	user := new(entities.User)
	err := h.DB.NewSelect().Model(user).Where("u.id = ?", sess.UserID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("用户不存在")
		}
		return nil, huma.Error500InternalServerError("查询用户失败")
	}

	id := uuid.NewV7()
	key := fmt.Sprintf("avatars/%v/%s%s", user.ID, id.String(), ".webp")

	img, err := image.ConvertAvatar(i.RawBody.Data().Image)
	if err != nil {
		return nil, huma.Error400BadRequest("转换头像失败")
	}

	err = h.Bucket.WriteAll(ctx, key, img, &blob.WriterOptions{
		ContentType: "image/webp",
	})
	if err != nil {
		slog.ErrorContext(ctx, "保存头像失败", "key", key, "error", err)
		return nil, huma.Error500InternalServerError("保存头像失败")
	}

	_, err = h.DB.NewUpdate().Model((*entities.User)(nil)).Set("image = ?", key).Where("id = ?", user.ID).Exec(ctx)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx), 5*time.Second,
		)
		defer cancel()

		if err := h.Bucket.Delete(cleanupCtx, key); err != nil {
			slog.ErrorContext(cleanupCtx, "清理头像失败", "key", key, "error", err)
		}

		slog.ErrorContext(ctx, "更新头像失败", "error", err)
		return nil, huma.Error500InternalServerError("更新头像失败")
	}

	return &struct{}{}, nil
}
