import type { Router } from 'vue-router'

import { useAuthStore } from '@/stores/auth'

export function setupRouterGuards(router: Router): void {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore()

    if (to.meta.requiresAuth) {
      await authStore.restore()
      if (!authStore.isLoggedIn) {
        return {
          path: '__DEFAULT_LOGIN_PATH__',
          query: { redirect: to.fullPath },
        }
      }
    }

    if (to.path === '__DEFAULT_LOGIN_PATH__') {
      await authStore.restore()
      if (authStore.isLoggedIn) {
        return '__DEFAULT_AUTH_REDIRECT__'
      }
    }

    return true
  })
}
