import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  { path: '/login', name: 'Login', component: () => import('@/views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue') },
      { path: 'upload', name: 'Upload', component: () => import('@/views/Upload.vue') },
      { path: 'gallery', name: 'Gallery', component: () => import('@/views/Gallery.vue') },
      { path: 'products', name: 'Products', component: () => import('@/views/Products.vue') },
      { path: 'styles', name: 'StylePresets', component: () => import('@/views/StylePresets.vue') },
      { path: 'models', name: 'ModelConfigs', component: () => import('@/views/ModelConfigs.vue') },
      { path: 'oss', name: 'OssConfig', component: () => import('@/views/OssConfig.vue') },
      { path: 'users', name: 'Users', component: () => import('@/views/Users.vue'), meta: { admin: true } },
      { path: 'logs', name: 'OperationLogs', component: () => import('@/views/OperationLogs.vue'), meta: { admin: true } }
    ]
  }
]

const router = createRouter({ history: createWebHashHistory(), routes })

router.beforeEach((to, from, next) => {
  const user = useUserStore()
  if (to.meta.public) return next()
  if (!user.isLogin) return next('/login')
  if (to.meta.admin && !user.isAdmin) return next('/dashboard')
  next()
})

export default router
