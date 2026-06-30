import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  { path: '/login', name: 'Login', component: () => import('@/views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      // 工作台（所有登录用户）
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/Dashboard.vue'), meta: { group: 'workspace' } },
      { path: 'upload', name: 'Upload', component: () => import('@/views/Upload.vue'), meta: { group: 'workspace' } },
      { path: 'products', name: 'Products', component: () => import('@/views/Products.vue'), meta: { group: 'workspace' } },
      { path: 'gallery', name: 'Gallery', component: () => import('@/views/Gallery.vue'), meta: { group: 'workspace' } },

      // 系统管理（仅管理员）
      { path: 'prompts', name: 'PromptSettings', component: () => import('@/views/PromptSettings.vue'), meta: { admin: true, group: 'admin' } },
      { path: 'models', name: 'ModelConfigs', component: () => import('@/views/ModelConfigs.vue'), meta: { admin: true, group: 'admin' } },
      { path: 'styles', name: 'StylePresets', component: () => import('@/views/StylePresets.vue'), meta: { admin: true, group: 'admin' } },
      { path: 'oss', name: 'OssConfig', component: () => import('@/views/OssConfig.vue'), meta: { admin: true, group: 'admin' } },
      { path: 'users', name: 'Users', component: () => import('@/views/Users.vue'), meta: { admin: true, group: 'admin' } },
      { path: 'logs', name: 'OperationLogs', component: () => import('@/views/OperationLogs.vue'), meta: { admin: true, group: 'admin' } },
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
