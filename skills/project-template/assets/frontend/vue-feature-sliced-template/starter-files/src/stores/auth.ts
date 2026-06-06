import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { getProfile } from '@/api/auth'
import type { AuthUser } from '@/entities/user/model/user'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const sessionRestored = ref(false)
  let restorePromise: Promise<void> | null = null

  const isLoggedIn = computed(() => !!user.value)

  function setAuth(nextUser: AuthUser): void {
    user.value = nextUser
    sessionRestored.value = true
  }

  function logout(): void {
    user.value = null
    sessionRestored.value = true
  }

  async function restore(): Promise<void> {
    if (user.value || sessionRestored.value) {
      return
    }
    if (restorePromise) {
      return restorePromise
    }

    restorePromise = (async () => {
      try {
        user.value = await getProfile()
      } catch {
        user.value = null
      } finally {
        sessionRestored.value = true
        restorePromise = null
      }
    })()

    return restorePromise
  }

  return {
    user,
    sessionRestored,
    isLoggedIn,
    setAuth,
    logout,
    restore,
  }
})
