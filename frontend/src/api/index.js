import http from './http'

export const authApi = {
  login: (data) => http.post('/auth/login', data),
  register: (data) => http.post('/auth/register', data),
  me: () => http.get('/auth/me'),
  changePassword: (data) => http.post('/auth/password', data)
}

export const productApi = {
  list: (params) => http.get('/products', { params }),
  create: (data) => http.post('/products', data),
  get: (id) => http.get(`/products/${id}`),
  update: (id, data) => http.put(`/products/${id}`, data),
  remove: (id) => http.delete(`/products/${id}`),
  checkName: (name) => http.get('/products/check-name', { params: { name } }),
}

export const sellingPointApi = {
  list: (params) => http.get('/selling-points', { params }),
  listByProduct: (id) => http.get(`/products/${id}/selling-points`),
  createForProduct: (id, data) => http.post(`/products/${id}/selling-points`, data),
  get: (id) => http.get(`/selling-points/${id}`),
  remove: (id) => http.delete(`/selling-points/${id}`)
}

export const imageApi = {
  list: (params) => http.get('/images', { params }),
  get: (id) => http.get(`/images/${id}`),
  remove: (id) => http.delete(`/images/${id}`)
}

export const galleryApi = {
  list: (params) => http.get('/gallery', { params }),
  get: (id) => http.get(`/gallery/${id}`),
  remove: (id) => http.delete(`/gallery/${id}`)
}

export const aiApi = {
  analyze: (formData) =>
    http.post('/ai/analyze', formData, { headers: { 'Content-Type': 'multipart/form-data' } }),
  generate: (data) => http.post('/ai/generate', data),
  taskStatus: (id) => http.get(`/ai/tasks/${id}`)
}

export const sourceImageApi = {
  // 给产品单张上传原图 + 自动 AI 解析（卖点 + prompt）。
  // 返回 { image, productId, imageCount, sellingPoints, prompt }。
  upload: (productId, file, modelConfigId) => {
    const fd = new FormData()
    fd.append('file', file)
    if (modelConfigId) fd.append('modelConfigId', String(modelConfigId))
    return http.post(`/products/${productId}/source-images`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  }
}

export const modelConfigApi = {
  list: () => http.get('/model-configs'),
  create: (data) => http.post('/model-configs', data),
  update: (id, data) => http.put(`/model-configs/${id}`, data),
  remove: (id) => http.delete(`/model-configs/${id}`)
}

export const ossConfigApi = {
  get: () => http.get('/oss-config'),
  update: (data) => http.put('/oss-config', data)
}

export const styleApi = {
  list: () => http.get('/style-presets'),
  create: (data) => http.post('/style-presets', data),
  update: (id, data) => http.put(`/style-presets/${id}`, data),
  remove: (id) => http.delete(`/style-presets/${id}`)
}

export const userApi = {
  list: () => http.get('/users'),
  create: (data) => http.post('/users', data),
  update: (id, data) => http.put(`/users/${id}`, data),
  remove: (id) => http.delete(`/users/${id}`)
}

export const logApi = {
  list: (params) => http.get('/operation-logs', { params }),
  stats: (params) => http.get('/operation-logs/stats', { params }),
}

export const promptSettingsApi = {
  get: () => http.get('/prompt-settings'),
  update: (data) => http.put('/prompt-settings', data),
  reset: () => http.post('/prompt-settings/reset'),
}
