import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import { setupRouterGuards } from './guards'
import { appShellRoute } from './routes/appShellRoute'
import { authRoutes } from './routes/authRoutes'
import { errorRoutes } from './routes/errorRoutes'

const routes: RouteRecordRaw[] = [...authRoutes, appShellRoute, ...errorRoutes]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

setupRouterGuards(router)

export default router
export { routes }
