import { createMiddleware } from "hono/factory"

type Bindings = {
    AUTH: Fetcher
}

type Variables = {
    userId: string
}

export const requireAuth = createMiddleware<{
    Bindings: Bindings
    Variables: Variables
}>(async (c, next) => {
    const url = new URL("/api/auth/get-session", c.req.url)

    const response = await c.env.AUTH.fetch(
        new Request(url, {
            method: "GET",
            headers: c.req.raw.headers,
        }),
    )

    if (!response.ok) {
        return c.json({ error: "认证失败" }, 401)
    }

    const session = await response.json<{
        user?: {
            id: string
        }
    } | null>()

    if (!session?.user?.id) {
        return c.json({ error: "未认证" }, 401)
    }

    c.set("userId", session.user.id)

    await next()
})