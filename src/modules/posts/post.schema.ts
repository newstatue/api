import {
    createInsertSchema,
    createUpdateSchema,
} from "drizzle-zod"
import { z } from "zod"
import { posts } from "../../db/schema"

const titleSchema = z.string().trim().min(1).max(200)
const contentSchema = z.string().min(1)

export const createPostSchema = createInsertSchema(posts)
    .pick({
        title: true,
        content: true,
    })
    .extend({
        title: titleSchema,
        content: contentSchema,
    })

export const updatePostSchema = createUpdateSchema(posts)
    .pick({
        title: true,
        content: true,
    })
    .extend({
        title: titleSchema.optional(),
        content: contentSchema.optional(),
    })
    .refine(
        (data) => Object.keys(data).length > 0,
        "至少需要提供一个要更新的字段",
    )