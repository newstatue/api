import {
    sqliteTable,
    text,
    integer, index,
} from "drizzle-orm/sqlite-core"

export const posts = sqliteTable("posts", {
    id: text("id").primaryKey().$defaultFn(()=>crypto.randomUUID()),

    authorId: text("author_id").notNull(),

    title: text("title").notNull(),

    content: text("content").notNull(),

    status: text("status", {
        enum: ["draft", "published"],
    }).notNull().default("draft"),

    createdAt: integer("created_at", {
        mode: "timestamp",
    }).notNull().$defaultFn(() => new Date()),

    publishedAt: integer("published_at", {
        mode: "timestamp",
    }),

    updatedAt: integer("updated_at", {
        mode: "timestamp",
    }).notNull().$onUpdate(() => new Date()),
}, (table) => [
    index("posts_author_idx").on(table.authorId)
])