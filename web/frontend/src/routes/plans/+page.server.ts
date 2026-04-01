import { fail, redirect } from "@sveltejs/kit"
import { env } from "$env/dynamic/private"
import type { Actions, PageServerLoad } from "./$types"

const apiBaseUrl = env.API_INTERNAL_BASE_URL || env.PUBLIC_API_BASE_URL || "http://localhost:8090"

type PlansResponse = {
  plans?: Array<{ months: number; price: number; currency: string }>
}

type SessionResponse = {
  authenticated: boolean
}

export const load: PageServerLoad = async ({ fetch, request }) => {
  try {
    const [plansResponse, sessionResponse] = await Promise.all([
      fetch(`${apiBaseUrl}/api/v1/plans`),
      fetch(`${apiBaseUrl}/api/v1/auth/session`, {
        headers: {
          cookie: request.headers.get("cookie") ?? ""
        }
      })
    ])

    const plansData = plansResponse.ok
      ? ((await plansResponse.json()) as PlansResponse)
      : { plans: [] }
    const sessionData = sessionResponse.ok
      ? ((await sessionResponse.json()) as SessionResponse)
      : { authenticated: false }

    return {
      plans: plansData.plans ?? [],
      authenticated: sessionData.authenticated
    }
  } catch {
    return {
      plans: [],
      authenticated: false
    }
  }
}

export const actions: Actions = {
  checkout: async ({ request, fetch }) => {
    const formData = await request.formData()
    const months = Number(formData.get("months") ?? 0)

    const response = await fetch(`${apiBaseUrl}/api/v1/checkout/yookassa`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        cookie: request.headers.get("cookie") ?? ""
      },
      body: JSON.stringify({ months })
    })

    if (response.status === 401) {
      throw redirect(303, "/login")
    }

    if (!response.ok) {
      return fail(response.status, {
        message: (await response.text()) || "Не удалось создать оплату"
      })
    }

    const data = (await response.json()) as { url: string }
    throw redirect(303, data.url)
  }
}
