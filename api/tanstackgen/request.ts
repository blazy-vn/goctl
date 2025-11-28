import type {
  MutationFunction,
  MutationOptions,
  QueryFunctionContext,
  QueryKey,
  QueryOptions,
} from '@tanstack/vue-query'
import { QueryClient } from '@tanstack/vue-query'
import { createCollection, type Collection, type CollectionConfig } from '@tanstack/db'
import {
  queryCollectionOptions,
  type QueryCollectionUtils,
} from '@tanstack/query-db-collection'

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

export function createApiClient(defaultConfig: ClientConfig = {}) {
  const baseConfig = { ...defaultConfig }

  const boundRequest = <
    TResponse,
    TBody = unknown,
    TParams = Record<string, unknown>,
    THeaders = Record<string, unknown>,
  >(
    args: RequestArgs<TBody, TParams, THeaders>,
    override?: ClientConfig,
  ) => request<TResponse, TBody, TParams, THeaders>(args, mergeConfig(baseConfig, override))

  return {
    request: boundRequest,
    withConfig: (override: ClientConfig) => createApiClient(mergeConfig(baseConfig, override)),
  }
}

export const defaultApiClient = createApiClient()

function mergeConfig(base: ClientConfig, override?: ClientConfig): ClientConfig {
  if (!override) return base
  return {
    ...base,
    ...override,
    headers: {
      ...(base.headers ?? {}),
      ...(override.headers ?? {}),
    },
  }
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60 * 1000,
      retry: 2,
      refetchOnWindowFocus: false,
      refetchOnReconnect: true,
    },
    mutations: {
      retry: 1,
    },
  },
})

export interface RequestCollectionConfig<
  TItem extends object,
  TResponse = Array<TItem>,
  TKey extends string | number = string | number,
  TBody = unknown,
  TParams = Record<string, unknown>,
  THeaders = Record<string, unknown>,
> {
  id?: string
  queryKey: QueryKey
  getKey: (item: TItem) => TKey
  request:
    | RequestArgs<TBody, TParams, THeaders>
    | ((context: QueryFunctionContext<QueryKey>) => RequestArgs<TBody, TParams, THeaders>)
  mapResponse?: (response: TResponse) => Array<TItem>
  enabled?: boolean
  staleTime?: number
  gcTime?: number
  retry?: number | false
  refetchInterval?: number | false
  meta?: Record<string, unknown>
  clientConfig?: ClientConfig
  collectionOptions?: Partial<
    Pick<
      CollectionConfig<TItem, TKey>,
      'onInsert' | 'onUpdate' | 'onDelete' | 'stringCollation'
    >
  >
  startSync?: boolean
}

export function createRequestCollection<
  TItem extends object,
  TResponse = Array<TItem>,
  TKey extends string | number = string | number,
  TBody = unknown,
  TParams = Record<string, unknown>,
  THeaders = Record<string, unknown>,
>(
  config: RequestCollectionConfig<TItem, TResponse, TKey, TBody, TParams, THeaders>,
): Collection<TItem, TKey, QueryCollectionUtils<TItem, TKey, TItem, unknown>> {
  const mapResponse =
    config.mapResponse ??
    ((response: TResponse) =>
      Array.isArray(response) ? (response as Array<TItem>) : ([] as Array<TItem>))

  const queryFn = async (context: QueryFunctionContext<QueryKey>) => {
    const requestArgs =
      typeof config.request === 'function'
        ? config.request(context)
        : config.request

    const response = await defaultApiClient.request<TResponse, TBody, TParams, THeaders>(
      {
        ...requestArgs,
        signal: context.signal,
      },
      config.clientConfig,
    )

    return mapResponse(response)
  }

  const collectionConfig = queryCollectionOptions({
    id: config.id,
    queryKey: config.queryKey,
    queryFn,
    queryClient,
    getKey: config.getKey,
    enabled: config.enabled ?? true,
    staleTime: config.staleTime,
    gcTime: config.gcTime,
    retry: config.retry,
    refetchInterval: config.refetchInterval,
    meta: config.meta,
    ...(config.collectionOptions ?? {}),
  })

  const collection = createCollection(collectionConfig)
  if (config.startSync) {
    void collection.preload()
  }
  return collection
}
