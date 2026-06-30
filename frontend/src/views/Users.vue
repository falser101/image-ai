<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <div>
        <el-button type="primary" @click="openCreate">新建员工</el-button>
        <span class="text-muted" style="margin-left:12px;font-size:12px">
          共 {{ list.length }} 个员工 · 账号创建后不可改
        </span>
      </div>
    </div>
    <el-table :data="list" stripe style="width:100%">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" label="账号" width="180" />
      <el-table-column prop="nickname" label="姓名" min-width="200" />
      <el-table-column prop="role" label="角色" width="120">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">{{ row.role === 'admin' ? '管理员' : '员工' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status === 'active' ? '正常' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="200">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button text type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button text type="danger" :disabled="row.id === 1" @click="remove(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="dlg" :title="form.id ? '编辑员工' : '新建员工'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="账号"><el-input v-model="form.username" :disabled="!!form.id" /></el-form-item>
        <el-form-item label="姓名"><el-input v-model="form.nickname" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role" style="width:200px">
            <el-option label="员工" value="employee" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:200px">
            <el-option label="正常" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item :label="form.id ? '重置密码' : '密码'">
          <el-input v-model="form.newPassword" show-password placeholder="留空则不修改" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { userApi } from '@/api'
import { formatDateTime } from '@/utils/format'
import { confirmDelete } from '@/utils/confirmDelete'

const list = ref([])
const dlg = ref(false)
const form = ref({ id: null, username: '', nickname: '', role: 'employee', status: 'active', newPassword: '' })
const load = async () => { list.value = await userApi.list() }
const openCreate = () => { form.value = { id: null, username: '', nickname: '', role: 'employee', status: 'active', newPassword: '' }; dlg.value = true }
const openEdit = (row) => { form.value = { ...row, newPassword: '' }; dlg.value = true }
const submit = async () => {
  if (!form.value.username || (!form.value.id && !form.value.newPassword)) { ElMessage.warning('请填写完整'); return }
  if (form.value.id) await userApi.update(form.value.id, form.value)
  else await userApi.create({ username: form.value.username, password: form.value.newPassword, nickname: form.value.nickname, role: form.value.role })
  dlg.value = false; ElMessage.success('已保存'); load()
}
const remove = async (row) => {
  try {
    await confirmDelete({
      label: '账号',
      expected: row.username,
      title: `将永久删除员工「${row.username}」`,
      description: '员工账号会被删除，其创建的产品 / 原图 / 生成图关联仍保留但不再归属该员工。',
    })
  } catch { return }
  await userApi.remove(row.id)
  ElMessage.success('已删除')
  load()
}
onMounted(load)
</script>

<style scoped>
.page-card :deep(.el-table .row-actions) {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  white-space: nowrap;
}
.page-card :deep(.el-table .row-actions .el-button) {
  padding: 0 6px;
}
.page-card :deep(.el-table .row-actions .el-button + .el-button) {
  margin-left: 0;
}
</style>
