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
		exists, err := tx.NewSelect().Model((*entities.User)(nil)).Where("u.id = ?", c.OwnerID).Exists(ctx)
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
	Body Community
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
		Body: Community{
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
		Where("cm.community_id = ?", c.ID).
		Where("cm.user_id = ?", s.UserID).
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

type CreateCommunityPostInput struct {
	Slug string `path:"slug"`
	Body CreateCommunityPost
}

type CreateCommunityPostOutput struct {
	Body Post
}

func (h *Handler) CreateCommunityPost(ctx context.Context, i *CreateCommunityPostInput) (*CreateCommunityPostOutput, error) {
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

	// 查询社区
	var community entities.Community
	err := h.DB.NewSelect().Model(&community).Where("c.slug = ?", i.Slug).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("社区不存在")
		}
		return nil, err
	}

	// 检查成员资格
	exists, err := h.DB.NewSelect().
		Model((*entities.CommunityMember)(nil)).
		Where("cm.community_id = ?", community.ID).
		Where("cm.user_id = ?", s.UserID).
		Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists && community.OwnerID != s.UserID {
		return nil, huma.Error403Forbidden("请先加入社区")
	}

	// 创建帖子
	post := &entities.CommunityPost{
		CommunityID: community.ID,
		AuthorID:    s.UserID,
		Title:       i.Body.Title,
		Content:     i.Body.Content,
	}

	_, err = h.DB.NewInsert().Model(post).Returning("*").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &CreateCommunityPostOutput{
		Body: Post{
			ID:          post.ID,
			CommunityID: post.CommunityID,
			AuthorID:    post.AuthorID,
			Title:       post.Title,
			Content:     post.Content,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		},
	}, nil
}

type ListCommunityPostsInput struct {
	Slug string `path:"slug"`
	pagination.Params
}

type ListCommunityPostsOutput struct {
	Body pagination.Result[Post]
}

func (h *Handler) ListCommunityPosts(ctx context.Context, i *ListCommunityPostsInput) (*ListCommunityPostsOutput, error) {
	// 查询社区
	var community entities.Community
	err := h.DB.NewSelect().
		Model(&community).
		Where("c.slug = ?", i.Slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound("社区不存在")
	}
	if err != nil {
		return nil, err
	}

	// 分页查询帖子
	query := h.DB.NewSelect().
		Model((*entities.CommunityPost)(nil)).
		Where("cp.community_id = ?", community.ID).
		OrderExpr("cp.created_at DESC, cp.id DESC")

	result, err := pagination.Query[entities.CommunityPost](ctx, query, i.Params)
	if err != nil {
		return nil, huma.Error500InternalServerError("查询帖子列表失败")
	}

	// 转换 DTO
	items := make([]Post, 0, len(result.Items))

	for _, post := range result.Items {
		items = append(items, Post{
			ID:          post.ID,
			CommunityID: post.CommunityID,
			AuthorID:    post.AuthorID,
			Title:       post.Title,
			Content:     post.Content,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	return &ListCommunityPostsOutput{
		Body: pagination.Result[Post]{
			Items: items,
			Total: result.Total,
			Page:  result.Page,
			Limit: result.Limit,
		},
	}, nil
}

type GetCommunityPostInput struct {
	Slug   string `path:"slug"`
	PostID string `path:"postId" format:"uuid"`
}

type GetCommunityPostOutput struct {
	Body Post
}

func (h *Handler) GetCommunityPost(ctx context.Context, i *GetCommunityPostInput) (*GetCommunityPostOutput, error) {
	var post entities.CommunityPost

	err := h.DB.NewSelect().
		Model(&post).
		Join("JOIN communities AS c ON c.id = cp.community_id").
		Where("cp.id = ?", i.PostID).
		Where("c.slug = ?", i.Slug).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound("帖子不存在")
	}
	if err != nil {
		return nil, err
	}

	return &GetCommunityPostOutput{
		Body: Post{
			ID:          post.ID,
			CommunityID: post.CommunityID,
			AuthorID:    post.AuthorID,
			Title:       post.Title,
			Content:     post.Content,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		},
	}, nil
}
