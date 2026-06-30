<template>
  <div class="page-card" style="max-width:760px">
    <el-form :model="form" label-width="120px" v-loading="loading">
      <el-form-item label="存储类型">
        <el-select v-model="form.provider" style="width:240px" @change="onProviderChange">
          <el-option label="本地存储 (local)" value="local" />
          <el-option label="阿里云 OSS" value="aliyun" />
          <el-option label="腾讯云 COS" value="tencent" />
          <el-option label="AWS S3" value="aws" />
        </el-select>
      </el-form-item>

      <!-- local：只读展示实际上传目录 -->
      <template v-if="isLocal">
        <el-form-item label="本地上传目录">
          <el-input :model-value="form.localDir" readonly>
            <template #append>
              <el-button @click="copyDir" :icon="DocumentCopy" />
            </template>
          </el-input>
          <div class="text-muted" style="font-size:12px;line-height:1.6;margin-top:4px">
            文件统一落在此目录（绝对路径，启动时由 <code>UPLOAD_DIR</code> 环境变量决定，默认 <code>./uploads</code>）。<br />
            当前实现 <b>只</b>使用本地存储：上方切换其他 provider 仅保存元信息，便于以后接入真 OSS 时平滑迁移。
          </div>
        </el-form-item>
      </template>

      <!-- 非 local：完整的 OSS 字段 -->
      <template v-else>
        <el-form-item label="Endpoint"><el-input v-model="form.endpoint" placeholder="oss-cn-hangzhou.aliyuncs.com" /></el-form-item>
        <el-form-item label="Bucket"><el-input v-model="form.bucket" /></el-form-item>
        <el-form-item label="Region"><el-input v-model="form.region" /></el-form-item>
        <el-form-item label="AccessKey"><el-input v-model="form.accessKey" show-password /></el-form-item>
        <el-form-item label="SecretKey"><el-input v-model="form.secretKey" show-password /></el-form-item>
        <el-form-item label="存储前缀"><el-input v-model="form.prefix" placeholder="image-ai/" /></el-form-item>
        <el-form-item label="公共访问域名"><el-input v-model="form.publicHost" placeholder="https://cdn.example.com" /></el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
          <span class="text-muted" style="margin-left:12px;font-size:12px">启用后将把新上传的文件保存到 OSS（当前实现保留本地存储逻辑可平滑切换）</span>
        </el-form-item>
      </template>

      <el-button type="primary" @click="save">保存</el-button>
    </el-form>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { DocumentCopy } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { ossConfigApi } from '@/api'

// 永远用一份空对象做表单初值，避免字段缺失
const emptyForm = () => ({
  provider: 'local',
  endpoint: '',
  bucket: '',
  accessKey: '',
  secretKey: '',
  region: '',
  prefix: '',
  publicHost: '',
  enabled: false,
  localDir: ''
})
const form = ref(emptyForm())
const loading = ref(false)

const isLocal = computed(() => form.value.provider === 'local')

const load = async () => {
  loading.value = true
  try {
    form.value = { ...emptyForm(), ...(await ossConfigApi.get()) }
  } finally {
    loading.value = false
  }
}

// 切到 local 时把 OSS 字段清掉，防止脏数据在表单里堆积；切到 OSS 时清掉 localDir 显示
const onProviderChange = (val) => {
  if (val === 'local') {
    form.value.endpoint = ''
    form.value.bucket = ''
    form.value.accessKey = ''
    form.value.secretKey = ''
    form.value.region = ''
    form.value.prefix = ''
    form.value.publicHost = ''
    form.value.enabled = false
  } else {
    form.value.localDir = ''
  }
}

const save = async () => {
  // 提交时按 provider 决定带哪些字段，避免给后端送没意义的数据
  const payload = isLocal.value
    ? { provider: 'local' }
    : { ...form.value }
  await ossConfigApi.update(payload)
  ElMessage.success('已保存')
  load()
}

const copyDir = async () => {
  if (!form.value.localDir) return
  try {
    await navigator.clipboard.writeText(form.value.localDir)
    ElMessage.success('已复制路径')
  } catch {
    ElMessage.warning('复制失败，请手动选中')
  }
}

onMounted(load)
</script>
