import { env } from "$env/dynamic/private"
import type { LayoutServerLoad } from "./$types"

const apiBaseUrl = env.API_INTERNAL_BASE_URL || env.PUBLIC_API_BASE_URL || "http://localhost:8090"

export const load: LayoutServerLoad = async ({ fetch, request }) => {
  try {
    const response = await fetch(`${apiBaseUrl}/api/v1/auth/session`, {
      headers: {
        cookie: request.headers.get("cookie") ?? ""
      }
    })

    const session = response.ok
      ? ((await response.json()) as { authenticated?: boolean })
      : { authenticated: false }

    return {
      authenticated: session.authenticated ?? false
    }
  } catch {
    return {
      authenticated: false
    }
  }
}
