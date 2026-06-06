import type { RouteRecordRaw } from 'vue-router'

export const errorRoutes: RouteRecordRaw[] = [
  {
    path: '/error/:status(\\d+)',
    name: 'ErrorStatus',
    component: () => import('@/pages/error/StatusRoutePage.vue'),
    props: (route) => ({
      status: Number(route.params.status),
    }),
  },
]
