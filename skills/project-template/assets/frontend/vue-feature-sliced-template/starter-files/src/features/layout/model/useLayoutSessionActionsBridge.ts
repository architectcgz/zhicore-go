import { logout as requestLogout } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

export function useLayoutSessionActionsBridge(onLoggedOut: () => void) {
  const authStore = useAuthStore()

  async function logout() {
    try {
      await requestLogout()
    } finally {
      authStore.logout()
    }
    onLoggedOut()
  }

  return { logout }
}
