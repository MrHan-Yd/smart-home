import { createRouter, createWebHistory } from 'vue-router'
import { fetchMe } from '@/api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/oauth/complete',
      name: 'oauth-complete',
      component: () => import('@/views/OAuthCompleteView.vue'),
      meta: { public: true },
    },
    {
      path: '/wall',
      name: 'wall',
      component: () => import('@/views/WallView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        {
          path: '',
          name: 'overview',
          component: () => import('@/views/OverviewView.vue'),
        },
        {
          path: 'devices',
          name: 'devices',
          component: () => import('@/views/DevicesView.vue'),
        },
        {
          path: 'devices/:id',
          name: 'device-detail',
          component: () => import('@/views/DeviceDetailView.vue'),
        },
        {
          path: 'rooms',
          name: 'rooms',
          component: () => import('@/views/RoomsView.vue'),
        },
        {
          path: 'add',
          name: 'add',
          component: () => import('@/views/AddDeviceView.vue'),
        },
        {
          path: 'logs',
          name: 'logs',
          component: () => import('@/views/ActionLogView.vue'),
        },
        {
          path: 'scenarios',
          name: 'scenarios',
          component: () => import('@/views/ScenariosView.vue'),
        },
        {
          path: 'analytics',
          name: 'analytics',
          component: () => import('@/views/AnalyticsView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
        },
      ],
    },
  ],
  scrollBehavior(to, from, saved) {
    if (saved) return saved
    if (to.path !== from.path) return { top: 0 }
    return undefined
  },
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  try {
    await fetchMe()
    return true
  } catch {
    const returnTo = to.fullPath || '/'
    window.location.href = `/oauth/login?return_to=${encodeURIComponent(returnTo)}`
    return false
  }
})

export default router
