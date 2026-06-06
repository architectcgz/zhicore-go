import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

export function useLoginForm() {
  const route = useRoute()
  const router = useRouter()
  const authStore = useAuthStore()

  const username = ref('')
  const password = ref('')
  const submitting = ref(false)
  const errorMessage = ref('')

  async function submit() {
    if (submitting.value) {
      return
    }

    errorMessage.value = ''
    submitting.value = true

    try {
      const user = await login({
        username: username.value,
        password: password.value,
      })
      authStore.setAuth(user)

      const redirect = typeof route.query.redirect === 'string'
        ? route.query.redirect
        : '__DEFAULT_AUTH_REDIRECT__'
      await router.push(redirect)
    } catch (error) {
      errorMessage.value = error instanceof Error ? error.message : '登录失败'
    } finally {
      submitting.value = false
    }
  }

  return {
    username,
    password,
    submitting,
    errorMessage,
    submit,
  }
}
