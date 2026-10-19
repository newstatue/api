export function getTrustedOrigins(
    env: {
        TRUSTED_ORIGINS?: string
    }
): string[] {
    return (env.TRUSTED_ORIGINS ?? "")
        .split(",")
        .map((origin) => origin.trim())
        .filter(Boolean)
}