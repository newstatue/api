package community

import (
	"context"
	"database/sql"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/newstatue/api/internal/pagination"
	"github.com/uptrace/bun"

	"github.com/newstatue/api/internal/entities"
	"github.com/newstatue/api/internal/session"
)

type Handler struct {
	DB      *bun.DB
	Session *session.Manager
}

type CreateCommunityInput struct {
	Body CreateCommunity
}

type CreateCommunityOutput struct {
	Body Community
}

func (h *Handler) CreateCommunity(ctx context.Context, i *CreateCommunityInput) (*CreateCommunityOutput, error) {
	// 1. 获取当前会话
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

	// 2. 检查平台管理员权限
	var user entities.User

	err := h.DB.NewSelect().Model(&user).Where("u.id = ?", s.UserID).Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error401Unauthorized("未认证")
	}
	if err != nil {
		return nil, err
	}

	if user.Role != entities.UserRoleAdmin {
		return nil, huma.Error403Forbidden("禁止操作")
	}

	// 3. 构造社区实体
	c := &entities.Community{
		Slug:        i.Body.Slug,
		Name:        i.Body.Name,
		Description: i.Body.Description,
		OwnerID:     i.Body.OwnerID,
	}

	// 4. 事务创建社区和 Owner 成员关系
	err = h.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// 检查 Owner 是否存在
		exists, err := tx.NewSelect().Model((*entities.User)(nil)).Where("id = ?", c.OwnerID).Exists(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return huma.Error422UnprocessableEntity("所有者不存在")
		}

		_, err = tx.NewInsert().Model(c).Exec(ctx)
		if err != nil {
			return err
		}

		member := &entities.CommunityMember{
			CommunityID: c.ID,
			UserID:      c.OwnerID,
			Role:        entities.CommunityRoleMember,
		}

		_, err = tx.NewInsert().Model(member).Exec(ctx)

		return err
	})
	if err != nil {
		return nil, err
	}

	community := Community{
		ID:          c.ID,
		Slug:        c.Slug,
		Name:        c.Name,
		Description: c.Description,
		OwnerID:     c.OwnerID,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	return &CreateCommunityOutput{
		Body: community,
	}, nil
}

type ListCommunityInput struct {
	pagination.Params
}

type ListCommunityOutput struct {
	Body pagination.Result[Community]
}

func (h *Handler) ListCommunities(ctx context.Context, i *ListCommunityInput) (*ListCommunityOutput, error) {
	query := h.DB.NewSelect().
		Model((*entities.Community)(nil)).
		OrderExpr("c.created_at DESC, c.id DESC")

	result, err := pagination.Query[entities.Community](ctx, query, i.Params)
	if err != nil {
		return nil, huma.Error500InternalServerError("查询社区列表失败")
	}

	items := make([]Community, 0, len(result.Items))

	for _, c := range result.Items {
		items = append(items, Community{
			ID:          c.ID,
			Slug:        c.Slug,
			Name:        c.Name,
			Description: c.Description,
			OwnerID:     c.OwnerID,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
	}

	return &ListCommunityOutput{
		Body: pagination.Result[Community]{
			Items: items,
			Total: result.Total,
			Page:  result.Page,
			Limit: result.Limit,
		},
	}, nil
}

type GetCommunityInput struct {
	Slug string `path:"slug"`
}

type GetCommunityOutput struct {
	Body *Community
}

func (h *Handler) GetCommunity(ctx context.Context, i *GetCommunityInput) (*GetCommunityOutput, error) {
	var c entities.Community

	err := h.DB.NewSelect().
		Model(&c).
		Where("c.slug = ?", i.Slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound("社区不存在")
	}
	if err != nil {
		return nil, err
	}

	return &GetCommunityOutput{
		Body: &Community{
			ID:          c.ID,
			Slug:        c.Slug,
			Name:        c.Name,
			Description: c.Description,
			OwnerID:     c.OwnerID,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		},
	}, nil
}

type JoinCommunityInput struct {
	Slug string `path:"slug"`
}

type JoinCommunityOutput struct {
	Body JoinCommunity
}

func (h *Handler) JoinCommunity(ctx context.Context, i *JoinCommunityInput) (*JoinCommunityOutput, error) {
	// 获取当前用户
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

	// 根据 Slug 查询社区
	var c entities.Community

	err := h.DB.NewSelect().
		Model(&c).
		Where("c.slug = ?", i.Slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound("社区不存在")
	}
	if err != nil {
		return nil, err
	}

	// 创建成员关系
	member := &entities.CommunityMember{
		CommunityID: c.ID,
		UserID:      s.UserID,
		Role:        entities.CommunityRoleMember,
	}

	_, err = h.DB.NewInsert().
		Model(member).
		On("CONFLICT (community_id, user_id) DO NOTHING").
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return &JoinCommunityOutput{
		Body: JoinCommunity{
			Joined: true,
		},
	}, nil
}

type LeaveCommunityInput struct {
	Slug string `path:"slug"`
}

type LeaveCommunityOutput struct {
	Body JoinCommunity
}

func (h *Handler) LeaveCommunity(ctx context.Context, i *LeaveCommunityInput) (*LeaveCommunityOutput, error) {
	// 获取当前用户
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

	// 根据 Slug 查询社区
	var c entities.Community

	err := h.DB.NewSelect().
		Model(&c).
		Where("c.slug = ?", i.Slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound("社区不存在")
	}
	if err != nil {
		return nil, err
	}

	// 社区所有者不能退出
	if c.OwnerID == s.UserID {
		return nil, huma.Error403Forbidden("社区所有者不能退出社区")
	}

	// 删除成员关系
	_, err = h.DB.NewDelete().
		Model((*entities.CommunityMember)(nil)).
		Where("community_id = ?", c.ID).
		Where("user_id = ?", s.UserID).
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return &LeaveCommunityOutput{
		Body: JoinCommunity{
			Joined: false,
		},
	}, nil
}
