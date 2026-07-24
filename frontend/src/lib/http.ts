export type ApiResult<T> = {
  code: number
  message: string
  data: T
}

export class ApiError extends Error {
  code: number
  httpStatus: number

  constructor(message: string, code: number, httpStatus: number) {
    super(message)
    this.code = code
    this.httpStatus = httpStatus
  }
}

let redirecting = false

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  opts?: { skipAuthRedirect?: boolean },
): Promise<ApiResult<T>> {
  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  })

  if (res.status === 401 && !opts?.skipAuthRedirect) {
    if (!redirecting) {
      redirecting = true
      const returnTo = encodeURIComponent(window.location.pathname + window.location.search)
      window.location.href = `/oauth/login?return_to=${returnTo}`
    }
    throw new ApiError('未登录', 40100, 401)
  }

  let json: ApiResult<T>
  try {
    json = (await res.json()) as ApiResult<T>
  } catch {
    throw new ApiError(res.statusText || '请求失败', 50000, res.status)
  }

  if (json.code !== 0) {
    throw new ApiError(json.message || '请求失败', json.code, res.status)
  }
  return json
}

export function apiGet<T>(path: string, opts?: { skipAuthRedirect?: boolean }) {
  return apiFetch<T>(path, { method: 'GET' }, opts)
}

export function apiPost<T>(path: string, body?: unknown, headers?: Record<string, string>) {
  return apiFetch<T>(path, {
    method: 'POST',
    body: body !== undefined ? JSON.stringify(body) : undefined,
    headers,
  })
}

export function apiPatch<T>(path: string, body: unknown) {
  return apiFetch<T>(path, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function apiDelete<T>(path: string) {
  return apiFetch<T>(path, { method: 'DELETE' })
}
