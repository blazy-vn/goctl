import type {
  MutationFunction,
  MutationOptions,
  QueryFunctionContext,
  QueryKey,
  QueryOptions,
} from '@tanstack/vue-query'

export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'OPTIONS'

export interface ClientConfig {
  baseUrl?: string
  headers?: Record<string, string>
  fetcher?: (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>
  signal?: AbortSignal
}

export interface RequestArgs<
  TBody = unknown,
  TParams = Record<string, unknown>,
  THeaders = Record<string, unknown>,
> {
  path: string
  method: Method
  body?: TBody
  params?: TParams
  headers?: THeaders
  signal?: AbortSignal
}

const paramRegex = /:([A-Za-z0-9_]+)/g

function toSearchParams(params: Record<string, unknown>): string {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null) return
    if (Array.isArray(value)) {
      value.forEach((item) => search.append(key, String(item)))
      return
    }
    search.append(key, String(value))
  })
  const queryString = search.toString()
  return queryString ? `?${queryString}` : ''
}

function buildUrl(
  path: string,
  params?: Record<string, unknown>,
): { url: string } {
  if (!params) return { url: path }

  const rest: Record<string, unknown> = { ...params }
  const url = path.replace(paramRegex, (_substring, key: string) => {
    const value = params[key]
    if (value === undefined || value === null) {
      return `:${key}`
    }
    delete rest[key]
    return encodeURIComponent(String(value))
  })

  return { url: `${url}${toSearchParams(rest)}` }
}

export class ApiError extends Error {
  status: number
  data: unknown

  constructor(status: number, data: unknown, message?: string) {
    super(message ?? `Request failed with status ${status}`)
    this.status = status
    this.data = data
  }
}

async function parseError(response: Response): Promise<ApiError> {
  let data: unknown
  try {
    data = await response.clone().json()
  } catch {
    try {
      data = await response.text()
    } catch {
      data = undefined
    }
  }

  // Type guard: after checking these conditions, TypeScript knows data is Record<string, unknown>
  const message =
    typeof data === 'object' && data !== null && 'message' in data
      ? String(data.message)  // No type assertion needed - TypeScript infers this correctly
      : response.statusText
  return new ApiError(response.status, data, message)
}

export async function request<
  TResponse,
  TBody = unknown,
  TParams = Record<string, unknown>,
  THeaders = Record<string, unknown>,
>(
  args: RequestArgs<TBody, TParams, THeaders>,
  config: ClientConfig = {},
): Promise<TResponse> {
  const { baseUrl = '', fetcher = fetch, headers: defaultHeaders } = config
  const { url } = buildUrl(args.path, args.params as Record<string, unknown>)

  const response = await fetcher(`${baseUrl}${url}`, {
    method: args.method,
    headers: {
      ...(args.body ? { 'Content-Type': 'application/json' } : {}),
      ...(defaultHeaders ?? {}),
      ...((args.headers as Record<string, string> | undefined) ?? {}),
    },
    body: args.body ? JSON.stringify(args.body) : undefined,
    credentials: 'include',
    signal: args.signal,
  })

  if (!response.ok) {
    throw await parseError(response)
  }

  if (response.status === 204) {
    return undefined as TResponse
  }

  return (await response.json()) as TResponse
}

export function createQueryOptions<TResponse>(
  queryKey: QueryKey,
  queryFn: (ctx: QueryFunctionContext) => Promise<TResponse>,
  options?: Partial<QueryOptions<TResponse, Error>>,
): QueryOptions<TResponse, Error> {
  return {
    queryKey,
    queryFn,
    ...(options ?? {}),
  }
}

export function createMutationOptions<TResponse, TVariables>(
  mutationFn: MutationFunction<TResponse, TVariables>,
  options?: Partial<MutationOptions<TResponse, Error, TVariables>>,
): MutationOptions<TResponse, Error, TVariables> {
  return {
    mutationFn,
    ...(options ?? {}),
  }
}
