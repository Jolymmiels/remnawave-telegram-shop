const API_BASE_URL =
  import.meta.env.PUBLIC_API_BASE_URL && import.meta.env.PUBLIC_API_BASE_URL.length > 0
    ? import.meta.env.PUBLIC_API_BASE_URL
    : "http://localhost:8090"

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    },
    ...init
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `Request failed with status ${response.status}`)
  }

  return response.json() as Promise<T>
}

export { API_BASE_URL }
