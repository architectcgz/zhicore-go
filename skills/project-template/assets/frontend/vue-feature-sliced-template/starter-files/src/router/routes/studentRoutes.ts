import type { RouteRecordRaw } from 'vue-router'

export const studentRoutes: RouteRecordRaw[] = [
  {
    path: 'student/dashboard',
    name: 'StudentDashboard',
    component: () => import('@/pages/dashboard/StudentDashboardRoutePage.vue'),
  },
]
