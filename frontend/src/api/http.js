import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import router from '@/router'

const http = axios.create({ baseURL: '/api', timeout: 120000 })

http.interceptors.request.use((cfg) => {
  const user = useUserStore()
  if (user.token) cfg.headers.Authorization = `Bearer ${user.token}`
  return cfg
})

http.interceptors.response.use(
  (resp) => {
    const data = resp.data
    if (data && data.code !== undefined && data.code !== 0) {
      ElMessage.error(data.message || '请求失败')
      return Promise.reject(data)
    }
    return data.data
  },
  (err) => {
    const status = err?.response?.status
    if (status === 401) {
      const user = useUserStore()
      user.logout()
      router.push('/login')
      ElMessage.error('登录已过期')
    } else {
      ElMessage.error(err?.response?.data?.message || err.message || '网络错误')
    }
    return Promise.reject(err)
  }
)

export default http
