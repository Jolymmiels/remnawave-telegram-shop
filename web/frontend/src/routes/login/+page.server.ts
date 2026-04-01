import { fail, redirect } from "@sveltejs/kit"
import { env } from "$env/dynamic/private"
import type { Actions, PageServerLoad } from "./$types"

const apiBaseUrl = env.API_INTERNAL_BASE_URL || env.PUBLIC_API_BASE_URL || "http://localhost:8090"
const sessionCookieName = "remnawave_session"

function extractSessionCookie(setCookieHeader: string | null) {
  if (!setCookieHeader) {
    return null
  }

  const firstPart = setCookieHeader.split(";")[0]
  const [name, value] = firstPart.split("=")

  if (name !== sessionCookieName || !value) {
    return null
  }

  return value
}

export const load: PageServerLoad = async ({ fetch, request }) => {
  try {
    const response = await fetch(`${apiBaseUrl}/api/v1/auth/session`, {
      headers: {
        cookie: request.headers.get("cookie") ?? ""
      }
    })
    const session = response.ok
      ? ((await response.json()) as { authenticated: boolean })
      : { authenticated: false }

    if (session.authenticated) {
      throw redirect(302, "/cabinet")
    }
  } catch {
    return {}
  }

  return {}
}

export const actions: Actions = {
  login: async ({ request, fetch, cookies }) => {
    const formData = await request.formData()
    const login = String(formData.get("login") ?? "").trim()
    const password = String(formData.get("password") ?? "")

    const response = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        identifier: login,
        password
      })
    })

    if (!response.ok) {
      return fail(response.status, {
        mode: "login",
        message: (await response.text()) || "Не удалось выполнить вход",
        values: { login }
      })
    }

    const sessionToken = extractSessionCookie(response.headers.get("set-cookie"))
    if (!sessionToken) {
      return fail(500, {
        mode: "login",
        message: "Не удалось создать сессию",
        values: { login }
      })
    }

    cookies.set(sessionCookieName, sessionToken, {
      path: "/",
      httpOnly: true,
      sameSite: "lax",
      secure: false,
      maxAge: 60 * 60 * 24 * 30
    })

    throw redirect(303, "/cabinet")
  },

  register: async ({ request, fetch, cookies }) => {
    const formData = await request.formData()
    const login = String(formData.get("login") ?? "").trim()
    const password = String(formData.get("password") ?? "")

    const response = await fetch(`${apiBaseUrl}/api/v1/auth/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        login,
        password,
        language: "ru"
      })
    })

    if (!response.ok) {
      return fail(response.status, {
        mode: "register",
        message: (await response.text()) || "Не удалось создать аккаунт",
        values: { login }
      })
    }

    const sessionToken = extractSessionCookie(response.headers.get("set-cookie"))
    if (!sessionToken) {
      return fail(500, {
        mode: "register",
        message: "Не удалось создать сессию",
        values: { login }
      })
    }

    cookies.set(sessionCookieName, sessionToken, {
      path: "/",
      httpOnly: true,
      sameSite: "lax",
      secure: false,
      maxAge: 60 * 60 * 24 * 30
    })

    throw redirect(303, "/cabinet")
  }
}
