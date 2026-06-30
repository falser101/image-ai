<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand gradient-text">AI 产品图生成</div>
      <el-menu :default-active="active" router :collapse="false" background-color="transparent" text-color="#c9d1d9" active-text-color="#fff">
        <el-menu-item index="/dashboard"><el-icon><DataBoard/></el-icon><span>工作台</span></el-menu-item>

        <el-sub-menu index="workspace">
          <template #title><el-icon><Box/></el-icon><span>工作区</span></template>
          <el-menu-item index="/upload"><el-icon><Upload/></el-icon><span>上传与生图</span></el-menu-item>
          <el-menu-item index="/products"><el-icon><Goods/></el-icon><span>产品</span></el-menu-item>
          <el-menu-item index="/gallery"><el-icon><Picture/></el-icon><span>图库</span></el-menu-item>
        </el-sub-menu>

        <el-sub-menu v-if="user.isAdmin" index="admin">
          <template #title><el-icon><Setting/></el-icon><span>系统管理</span></template>
          <el-menu-item index="/prompts"><el-icon><ChatDotRound/></el-icon><span>提示词配置</span></el-menu-item>
          <el-menu-item index="/models"><el-icon><Connection/></el-icon><span>模型配置</span></el-menu-item>
          <el-menu-item index="/styles"><el-icon><Brush/></el-icon><span>风格预设</span></el-menu-item>
          <el-menu-item index="/oss"><el-icon><Coin/></el-icon><span>OSS 配置</span></el-menu-item>
          <el-menu-item index="/users"><el-icon><User/></el-icon><span>员工管理</span></el-menu-item>
          <el-menu-item index="/logs"><el-icon><Document/></el-icon><span>操作日志</span></el-menu-item>
        </el-sub-menu>
      </el-menu>
    </aside>
    <section class="content">
      <header class="topbar">
        <div class="flex items-center gap-12">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="flex items-center gap-12">
          <el-tag :type="user.isAdmin ? 'danger' : 'primary'" effect="dark">{{ user.isAdmin ? '管理员' : '员工' }}</el-tag>
          <el-dropdown @command="onCommand">
            <span class="flex items-center gap-8" style="cursor:pointer">
              <el-avatar :size="28">{{ (user.user?.nickname || user.user?.username || 'U').charAt(0) }}</el-avatar>
              <span>{{ user.user?.nickname || user.user?.username }}</span>
              <el-icon><ArrowDown/></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      <main class="main">
        <router-view />
      </main>
    </section>

    <el-dialog v-model="pwdVisible" title="修改密码" width="420px">
      <el-form :model="pwdForm" label-width="80px">
        <el-form-item label="原密码"><el-input v-model="pwdForm.old" type="password" show-password /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="pwdForm.new" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" @click="submitPwd">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import { authApi } from '@/api'

const route = useRoute()
const router = useRouter()
const user = useUserStore()
const active = computed(() => route.path)
const titles = {
  '/dashboard': '工作台',
  '/upload': '上传与生图',
  '/gallery': '图库',
  '/products': '产品',
  '/prompts': '提示词配置',
  '/styles': '风格预设',
  '/models': '模型配置',
  '/oss': 'OSS 配置',
  '/users': '员工管理',
  '/logs': '操作日志'
}
const pageTitle = computed(() => titles[route.path] || '')

const pwdVisible = ref(false)
const pwdForm = ref({ old: '', new: '' })
const onCommand = (cmd) => {
  if (cmd === 'logout') { user.logout(); router.push('/login') }
  if (cmd === 'password') pwdVisible.value = true
}
const submitPwd = async () => {
  if (!pwdForm.value.old || pwdForm.value.new.length < 6) {
    ElMessage.warning('请填写原密码与至少6位新密码'); return
  }
  await authApi.changePassword(pwdForm.value)
  ElMessage.success('密码已修改')
  pwdVisible.value = false
}
</script>
