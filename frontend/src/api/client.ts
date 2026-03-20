const BASE = import.meta.env.VITE_API_BASE ?? ''

export class ApiError extends Error {
  constructor(public code: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('access_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }
  if (token) headers['Authorization'] = 'Bearer ' + token

  const res  = await fetch(BASE + path, { ...options, headers })
  const json = await res.json()

  if (json.code !== 0) {
    throw new ApiError(json.code, json.message ?? '请求失败')
  }
  return json.data as T
}

export const api = {
  get:    <T>(path: string)                => request<T>(path, { method: 'GET' }),
  post:   <T>(path: string, body: unknown) => request<T>(path, { method: 'POST',  body: JSON.stringify(body) }),
  put:    <T>(path: string, body: unknown) => request<T>(path, { method: 'PUT',   body: JSON.stringify(body) }),
  delete: <T>(path: string)               => request<T>(path, { method: 'DELETE' }),
}
