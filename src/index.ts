import { Hono } from "hono"
import { cors } from "hono/cors"
import {getTrustedOrigins} from "./config";

type Bindings = {
  AUTH: Fetcher
}

const app = new Hono<{ Bindings: Bindings }>()

app.use("/api/*",cors({
  origin: (origin, c)=>{
    const trustedOrigins = getTrustedOrigins(c.env)

    return trustedOrigins.includes(origin) ? origin : undefined
  },
  allowMethods: [ "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
  allowHeaders: ["Content-Type", "Authorization"],
  credentials: true,
}) )

app.all("/api/auth/*", (c) => {
  return c.env.AUTH.fetch(c.req.raw)
})

export default app