import type { App, ComponentPublicInstance } from 'vue'
import type { Pinia } from 'pinia'
import type { Router } from 'vue-router'

import { ApiError, getAxiosInstance } from '@/api/request'
import { useAuthStore } from '@/stores/auth'

let httpErrorHandlingInstalled = false

export interface ErrorRuntimeOptions {
  loginPath?: string
  errorStatusPath?: (status: number) => string
  handleUnauthorized?: (context: {
    router: Router
    pinia: Pinia
    error: ApiError
  }) => void | Promise<void>
  handleVueError?: (context: {
    router: Router
    error: unknown
    info: string
  }) => void | Promise<void>
  handleRouteError?: (context: {
    router: Router
    error: unknown
  }) => void | Promise<void>
}

function resolveAuthStore(pinia?: Pinia) {
  return pinia ? useAuthStore(pinia) : useAuthStore()
}

function withDefaultErrorRuntimeOptions(options: ErrorRuntimeOptions = {}): Required<ErrorRuntimeOptions> {
  const loginPath = options.loginPath || '__DEFAULT_LOGIN_PATH__'
  const errorStatusPath = options.errorStatusPath || ((status: number) => `/error/${status}`)

  return {
    loginPath,
    errorStatusPath,
    handleUnauthorized: options.handleUnauthorized || (async ({ router, pinia }) => {
      const authStore = resolveAuthStore(pinia)
      authStore.logout()
      await router.push(loginPath)
    }),
    handleVueError: options.handleVueError || (async ({ router }) => {
      await router.push(errorStatusPath(500))
    }),
    handleRouteError: options.handleRouteError || (async ({ router }) => {
      await router.push(errorStatusPath(500))
    }),
  }
}

export function createDefaultErrorRuntimeOptions(): ErrorRuntimeOptions {
  return withDefaultErrorRuntimeOptions()
}

export function installGlobalHttpErrorHandling(
  router: Router,
  pinia: Pinia,
  options: ErrorRuntimeOptions
): void {
  if (httpErrorHandlingInstalled) {
    return
  }

  getAxiosInstance().interceptors.response.use(
    (response) => response,
    (error: unknown) => {
      if (error instanceof ApiError && error.status === 401) {
        const normalized = withDefaultErrorRuntimeOptions(options)
        void Promise.resolve(normalized.handleUnauthorized({ router, pinia, error }))
      }
      return Promise.reject(error)
    }
  )

  httpErrorHandlingInstalled = true
}

export function createGlobalVueErrorHandler(router: Router, options: ErrorRuntimeOptions) {
  return (
    err: unknown,
    _instance: ComponentPublicInstance | null,
    info: string
  ): void => {
    console.error('Vue error:', err, info)
    if (err instanceof ApiError) {
      return
    }
    const normalized = withDefaultErrorRuntimeOptions(options)
    void Promise.resolve(normalized.handleVueError({ router, error: err, info }))
  }
}

export function createGlobalRouterErrorHandler(router: Router, options: ErrorRuntimeOptions) {
  return (error: unknown): void => {
    console.error('Router error:', error)
    const normalized = withDefaultErrorRuntimeOptions(options)
    void Promise.resolve(normalized.handleRouteError({ router, error }))
  }
}

export function setupGlobalErrorRuntime(
  app: App,
  router: Router,
  pinia: Pinia,
  options: ErrorRuntimeOptions = createDefaultErrorRuntimeOptions()
): void {
  installGlobalHttpErrorHandling(router, pinia, options)
  app.config.errorHandler = createGlobalVueErrorHandler(router, options)
  router.onError(createGlobalRouterErrorHandler(router, options))
}
