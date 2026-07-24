export type ApiBody<T = unknown> = {
  code: number
  message: string
  data?: T
}

export async function apiGet<T>(path: string): Promise<ApiBody<T>> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  return (await res.json()) as ApiBody<T>
}
