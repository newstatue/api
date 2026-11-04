package community

import (
	"context"
	"database/sql"
	"errors"
	"uuid"

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

type CreateInput struct {
	Body Create
}

type CreateOutput struct {
	Body Community
}

func (h *Handler) Create(ctx context.Context, i *CreateInput) (*CreateOutput, error) {
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
			Role:        entities.CommunityRoleOwner,
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

	return &CreateOutput{
		Body: community,
	}, nil
}

type ListInput struct {
	pagination.Params
}

type ListOutput struct {
	Body pagination.Result[Community]
}

func (h *Handler) List(ctx context.Context, i *ListInput) (*ListOutput, error) {
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

	return &ListOutput{
		Body: pagination.Result[Community]{
			Items: items,
			Total: result.Total,
			Page:  result.Page,
			Limit: result.Limit,
		},
	}, nil
}

type GetInput struct {
	Slug string `path:"slug"`
}

type GetOutput struct {
	Body Community
}

func (h *Handler) Get(ctx context.Context, i *GetInput) (*GetOutput, error) {
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

	r := Community{
		ID:          c.ID,
		Slug:        c.Slug,
		Name:        c.Name,
		Description: c.Description,
		OwnerID:     c.OwnerID,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	if sess, ok := session.FromContext(ctx); ok {
		var member entities.CommunityMember

		err := h.DB.NewSelect().
			Model(&member).
			Where("cm.community_id = ?", c.ID).
			Where("cm.user_id = ?", sess.UserID).
			Scan(ctx)

		if err == nil {
			r.IsMember = true
			r.MemberRole = string(member.Role)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error500InternalServerError("查询社区成员状态失败")
		}
	}

	return &GetOutput{
		Body: r,
	}, nil
}

type PutMembershipInput struct {
	Slug string `path:"slug"`
	Body struct {
		Joined bool `json:"joined"`
	}
}

func (h *Handler) PutMembership(ctx context.Context, i *PutMembershipInput) (*struct{}, error) {
	s, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("未认证")
	}

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

	if i.Body.Joined {
		member := &entities.CommunityMember{
			CommunityID: c.ID,
			UserID:      s.UserID,
			Role:        entities.CommunityRoleMember,
		}
		_, err = h.DB.NewInsert().
			Model(member).
			On("CONFLICT (community_id, user_id) DO NOTHING").
			Exec(ctx)
	} else {
		if c.OwnerID == s.UserID {
			return nil, huma.Error403Forbidden("社区所有者不能退出社区")
		}
		_, err = h.DB.NewDelete().
			Model((*entities.CommunityMember)(nil)).
			Where("cm.community_id = ?", c.ID).
			Where("cm.user_id = ?", s.UserID).
			Exec(ctx)
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}

type CreatePostInput struct {
	Slug string `path:"slug"`
	Body CreatePost
}

type CreatePostOutput struct {
	Body Post
}

func (h *Handler) CreatePost(ctx context.Context, i *CreatePostInput) (*CreatePostOutput, error) {
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

	return &CreatePostOutput{
		Body: Post{
			VoteCount:   post.VoteCount,
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

type ListPostsInput struct {
	Slug string `path:"slug"`
	pagination.Params
}

type ListPostsOutput struct {
	Body pagination.Result[Post]
}

func (h *Handler) ListPosts(ctx context.Context, i *ListPostsInput) (*ListPostsOutput, error) {
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
		Column("cp.*").
		ColumnExpr("(SELECT COALESCE(SUM(cpv.value), 0) FROM community_post_votes AS cpv WHERE cpv.post_id = cp.id) AS vote_count").
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
			VoteCount:   post.VoteCount,
			ID:          post.ID,
			CommunityID: post.CommunityID,
			AuthorID:    post.AuthorID,
			Title:       post.Title,
			Content:     post.Content,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
		})
	}

	return &ListPostsOutput{
		Body: pagination.Result[Post]{
			Items: items,
			Total: result.Total,
			Page:  result.Page,
			Limit: result.Limit,
		},
	}, nil
}

type GetPostInput struct {
	Slug   string `path:"slug"`
	PostID string `path:"postId" format:"uuid"`
}

type GetPostOutput struct {
	Body Post
}

func (h *Handler) GetPost(ctx context.Context, i *GetPostInput) (*GetPostOutput, error) {
	var post entities.CommunityPost

	err := h.DB.NewSelect().
		Model(&post).
		Column("cp.*").
		ColumnExpr("(SELECT COALESCE(SUM(cpv.value), 0) FROM community_post_votes AS cpv WHERE cpv.post_id = cp.id) AS vote_count").
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

	return &GetPostOutput{
		Body: Post{
			VoteCount:   post.VoteCount,
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

type PutPostVoteInput struct {
	PostID uuid.UUID `path:"postId"`
	Body   PutPostVote
}

func (h *Handler) PutPostVote(ctx context.Context, i *PutPostVoteInput) (*struct{}, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("请先登录")
	}

	vote := &entities.CommunityPostVote{
		PostID: i.PostID,
		UserID: sess.UserID,
		Value:  i.Body.Value,
	}

	if vote.Value == 0 {
		_, err := h.DB.NewDelete().Model(vote).WherePK().Exec(ctx)

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err := h.DB.NewInsert().
		Model(vote).
		On("CONFLICT (post_id, user_id) DO UPDATE").
		Set("value = EXCLUDED.value").
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

type PutPostBookmarkInput struct {
	PostID uuid.UUID `path:"postId"`
	Body   PutPostBookmark
}

func (h *Handler) PutPostBookmark(ctx context.Context, i *PutPostBookmarkInput) (*struct{}, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("请先登录")
	}

	bookmark := &entities.CommunityPostBookmark{
		PostID: i.PostID,
		UserID: sess.UserID,
	}

	if !i.Body.Bookmarked {
		_, err := h.DB.NewDelete().
			Model(bookmark).
			WherePK().
			Exec(ctx)

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err := h.DB.NewInsert().
		Model(bookmark).
		On("CONFLICT (post_id, user_id) DO NOTHING").
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return nil, nil
}
