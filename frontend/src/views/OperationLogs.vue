<template>
  <div class="page-card">
    <div class="flex items-center gap-12 mb-12">
      <el-input v-model="q.keyword" placeholder="搜索资源/详情/账号" clearable style="width:240px" @keyup.enter="load" />
      <el-select v-model="q.action" placeholder="动作" clearable style="width:140px" @change="load">
        <el-option label="POST" value="POST" />
        <el-option label="PUT" value="PUT" />
        <el-option label="DELETE" value="DELETE" />
        <el-option label="LOGIN" value="LOGIN" />
      </el-select>
      <el-button @click="load">查询</el-button>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" label="账号" width="120" />
      <el-table-column prop="action" label="动作" width="100" />
      <el-table-column prop="resource" label="资源" width="220" />
      <el-table-column prop="detail" label="详情" show-overflow-tooltip />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="createdAt" label="时间" width="180" />
    </el-table>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { logApi } from '@/api'
const list = ref([])
const q = ref({ keyword: '', action: '' })
const load = async () => { list.value = await logApi.list(q.value) }
onMounted(load)
</script>
