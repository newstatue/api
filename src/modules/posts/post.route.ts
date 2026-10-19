import { Hono } from "hono"
import { zValidator } from "@hono/zod-validator"
import { drizzle } from "drizzle-orm/d1"
import { and, desc, eq } from "drizzle-orm"

import { posts } from "../../db/schema"
import { requireAuth } from "../../middleware/auth"
import {
    createPostSchema,
    updatePostSchema,
} from "./post.schema"

type Env = {
    Bindings: {
        DB: D1Database
        AUTH: Fetcher
    }
    Variables: {
        userId: string
    }
}

export const postsRoute = new Hono<Env>()

// CREATE - 创建文章草稿
postsRoute.post(
    "/",
    requireAuth,
    zValidator("json", createPostSchema),
    async (c) => {
        const db = drizzle(c.env.DB)
        const data = c.req.valid("json")
        const authorId = c.get("userId")

        const [post] = await db
            .insert(posts)
            .values({
                ...data,
                authorId,
                status: "draft",
            })
            .returning()

        return c.json(post, 201)
    },
)

// READ ALL - 获取已发布文章
postsRoute.get("/", async (c) => {
    const db = drizzle(c.env.DB)

    const result = await db
        .select()
        .from(posts)
        .where(eq(posts.status, "published"))
        .orderBy(desc(posts.publishedAt))
        .limit(20)

    return c.json(result)
})

// READ MINE - 获取当前用户所有文章
// 注意：必须放在 /:id 之前
postsRoute.get("/me", requireAuth, async (c) => {
    const db = drizzle(c.env.DB)
    const userId = c.get("userId")

    const result = await db
        .select()
        .from(posts)
        .where(eq(posts.authorId, userId))
        .orderBy(desc(posts.createdAt))
        .limit(100)

    return c.json(result)
})

// READ ONE - 获取已发布文章详情
postsRoute.get("/:id", async (c) => {
    const db = drizzle(c.env.DB)
    const id = c.req.param("id")

    const [post] = await db
        .select()
        .from(posts)
        .where(
            and(
                eq(posts.id, id),
                eq(posts.status, "published"),
            ),
        )
        .limit(1)

    if (!post) {
        return c.json({ error: "文章未找到" }, 404)
    }

    return c.json(post)
})

// UPDATE - 修改自己的文章
postsRoute.patch(
    "/:id",
    requireAuth,
    zValidator("json", updatePostSchema),
    async (c) => {
        const db = drizzle(c.env.DB)
        const id = c.req.param("id")
        const userId = c.get("userId")
        const data = c.req.valid("json")

        const [post] = await db
            .update(posts)
            .set(data)
            .where(
                and(
                    eq(posts.id, id),
                    eq(posts.authorId, userId),
                ),
            )
            .returning()

        if (!post) {
            return c.json({ error: "文章未找到" }, 404)
        }

        return c.json(post)
    },
)

// DELETE - 删除自己的文章
postsRoute.delete("/:id", requireAuth, async (c) => {
    const db = drizzle(c.env.DB)
    const id = c.req.param("id")
    const userId = c.get("userId")

    const [post] = await db
        .delete(posts)
        .where(
            and(
                eq(posts.id, id),
                eq(posts.authorId, userId),
            ),
        )
        .returning()

    if (!post) {
        return c.json({ error: "文章未找到" }, 404)
    }

    return c.body(null, 204)
})

// PUBLISH - 发布自己的草稿
postsRoute.post(
    "/:id/publish",
    requireAuth,
    async (c) => {
        const db = drizzle(c.env.DB)
        const id = c.req.param("id")
        const userId = c.get("userId")

        const [post] = await db
            .update(posts)
            .set({
                status: "published",
                publishedAt: new Date(),
            })
            .where(
                and(
                    eq(posts.id, id),
                    eq(posts.authorId, userId),
                    eq(posts.status, "draft"),
                ),
            )
            .returning()

        if (!post) {
            return c.json({
                error: "文章未找到或已发布",
            }, 404)
        }

        return c.json(post)
    },
)
