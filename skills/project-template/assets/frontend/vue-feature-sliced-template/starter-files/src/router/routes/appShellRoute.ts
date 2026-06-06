import type { RouteRecordRaw } from 'vue-router'

import { studentRoutes } from './studentRoutes'

export const appShellRoute: RouteRecordRaw = {
  path: '/',
  component: () => import('@/pages/AppShellRoutePage.vue'),
  redirect: '__DEFAULT_AUTH_REDIRECT__',
  meta: { requiresAuth: true },
  children: [...studentRoutes],
}
