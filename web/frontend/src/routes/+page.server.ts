import { env } from "$env/dynamic/private"
import type { PageServerLoad } from "./$types"

const apiBaseUrl = env.API_INTERNAL_BASE_URL || env.PUBLIC_API_BASE_URL || "http://localhost:8090"

export const load: PageServerLoad = async ({ fetch }) => {
  try {
    const response = await fetch(`${apiBaseUrl}/api/v1/plans`)
    const data = (await response.json()) as {
      plans?: Array<{ months: number; price: number; currency: string }>
      trialDays?: number
    }

    return {
      plans: data.plans ?? [],
      trialDays: data.trialDays ?? 0
    }
  } catch {
    return {
      plans: [],
      trialDays: 0
    }
  }
}
