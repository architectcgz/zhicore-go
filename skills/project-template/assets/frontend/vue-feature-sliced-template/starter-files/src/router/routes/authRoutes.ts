import type { RouteRecordRaw } from 'vue-router'

export const authRoutes: RouteRecordRaw[] = [
  {
    path: '__DEFAULT_LOGIN_PATH__',
    name: 'Login',
    component: () => import('@/pages/auth/LoginRoutePage.vue'),
  },
]
