<template>
  <div class="page-card" style="max-width:760px">
    <el-form :model="form" label-width="120px" v-loading="loading">
      <el-form-item label="Provider">
        <el-select v-model="form.provider" style="width:240px">
          <el-option label="本地存储 (local)" value="local" />
          <el-option label="阿里云 OSS" value="aliyun" />
          <el-option label="腾讯云 COS" value="tencent" />
          <el-option label="AWS S3" value="aws" />
        </el-select>
      </el-form-item>
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
      <el-button type="primary" @click="save">保存</el-button>
    </el-form>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ossConfigApi } from '@/api'

const form = ref({ provider: 'local', endpoint: '', bucket: '', accessKey: '', secretKey: '', region: '', prefix: '', publicHost: '', enabled: false })
const loading = ref(false)
const load = async () => {
  loading.value = true
  try { form.value = { ...form.value, ...(await ossConfigApi.get()) } }
  finally { loading.value = false }
}
const save = async () => {
  await ossConfigApi.update(form.value)
  ElMessage.success('已保存'); load()
}
onMounted(load)
</script>
