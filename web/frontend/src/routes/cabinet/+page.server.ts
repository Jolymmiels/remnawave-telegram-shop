import { fail, redirect } from "@sveltejs/kit"
import { env } from "$env/dynamic/private"
import type { Actions, PageServerLoad } from "./$types"

type SessionResponse = {
  authenticated: boolean
  customer?: {
    id: number
    telegramId?: number | null
    login?: string | null
    language?: string
    subscriptionLink?: string | null
    expireAt?: string | null
  }
  purchase?: {
    id: number
    status: "new" | "pending" | "paid" | "cancel"
    invoiceType: string
    amount: number
    currency: string
    months: number
    createdAt: string
    paidAt?: string | null
    paymentUrl?: string | null
  } | null
  purchases?: Array<{
    id: number
    status: "new" | "pending" | "paid" | "cancel"
    invoiceType: string
    amount: number
    currency: string
    months: number
    createdAt: string
    paidAt?: string | null
    paymentUrl?: string | null
  }>
  trialAvailable?: boolean
}

const apiBaseUrl = env.API_INTERNAL_BASE_URL || env.PUBLIC_API_BASE_URL || "http://localhost:8090"
const sessionCookieName = "remnawave_session"

async function loadSession(fetchFn: typeof fetch, cookieHeader: string) {
  const response = await fetchFn(`${apiBaseUrl}/api/v1/auth/session`, {
    headers: {
      cookie: cookieHeader
    }
  })

  if (!response.ok) {
    return {
      authenticated: false,
      customer: undefined,
      purchase: null,
      purchases: [],
      trialAvailable: false
    } satisfies SessionResponse
  }

  const session = (await response.json()) as SessionResponse
  return {
    authenticated: session.authenticated,
    customer: session.customer,
    purchase: session.purchase ?? null,
    purchases: session.purchases ?? [],
    trialAvailable: session.trialAvailable ?? false
  } satisfies SessionResponse
}

export const load: PageServerLoad = async ({ fetch, request }) => {
  try {
    return await loadSession(fetch, request.headers.get("cookie") ?? "")
  } catch {
    return {
      authenticated: false,
      customer: undefined,
      purchase: null,
      purchases: [],
      trialAvailable: false
    } satisfies SessionResponse
  }
}

export const actions: Actions = {
  refresh: async () => {
    return { refreshed: true }
  },

  logout: async ({ fetch, cookies, request }) => {
    await fetch(`${apiBaseUrl}/api/v1/auth/logout`, {
      method: "POST",
      headers: {
        cookie: request.headers.get("cookie") ?? ""
      }
    })

    cookies.delete(sessionCookieName, {
      path: "/"
    })

    throw redirect(303, "/login")
  },

  activateTrial: async ({ fetch, request }) => {
    const response = await fetch(`${apiBaseUrl}/api/v1/trial/activate`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        cookie: request.headers.get("cookie") ?? ""
      },
      body: "{}"
    })

    if (response.status === 401) {
      throw redirect(303, "/login")
    }

    if (!response.ok) {
      return fail(response.status, {
        message: (await response.text()) || "Не удалось активировать пробный период"
      })
    }

    return {
      success: true
    }
  },

  linkTelegram: async ({ fetch, request }) => {
    const response = await fetch(`${apiBaseUrl}/api/v1/auth/link-telegram`, {
      method: "POST",
      headers: {
        cookie: request.headers.get("cookie") ?? ""
      }
    })

    if (response.status === 401) {
      throw redirect(303, "/login")
    }

    if (response.status === 409) {
      return fail(response.status, {
        message: "Telegram уже привязан к этому аккаунту или возник конфликт привязки."
      })
    }

    if (response.status === 422) {
      return fail(response.status, {
        message:
          "Этот Telegram уже связан с другим аккаунтом, а текущий профиль нельзя безопасно объединить автоматически."
      })
    }

    if (!response.ok) {
      return fail(response.status, {
        message: (await response.text()) || "Не удалось создать ссылку для привязки Telegram"
      })
    }

    const data = (await response.json()) as { url: string }
    throw redirect(303, data.url)
  }
}
