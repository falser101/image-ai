<template>
  <div class="login-bg">
    <div class="login-card">
      <h2><span class="gradient-text">AI 产品图生成平台</span></h2>
      <el-form :model="form" label-width="0" @keyup.enter="submit">
        <el-form-item>
          <el-input v-model="form.username" placeholder="账号" size="large" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" show-password placeholder="密码" size="large" prefix-icon="Lock" />
        </el-form-item>
        <el-button type="primary" size="large" style="width:100%" :loading="loading" @click="submit">登录</el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import { authApi } from '@/api'

const router = useRouter()
const user = useUserStore()
const loading = ref(false)
const form = ref({ username: '', password: '' })
const submit = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请填写账号密码'); return
  }
  loading.value = true
  try {
    const data = await authApi.login(form.value)
    user.setAuth(data.token, data.user)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } finally { loading.value = false }
}
</script>
