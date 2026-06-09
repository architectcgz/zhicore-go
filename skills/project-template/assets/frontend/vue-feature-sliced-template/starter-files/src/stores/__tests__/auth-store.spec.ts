import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAuthStore } from '@/stores/auth'
import * as authApi from '@/api/auth'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('restores session only once after a successful profile fetch', async () => {
    const getProfile = vi.spyOn(authApi, 'getProfile').mockResolvedValue({
      id: 'user-1',
      username: 'student.demo',
      role: 'student',
    })

    const store = useAuthStore()
    await store.restore()
    await store.restore()

    expect(getProfile).toHaveBeenCalledTimes(1)
    expect(store.isLoggedIn).toBe(true)
    expect(store.sessionRestored).toBe(true)
  })
})
