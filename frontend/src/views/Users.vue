<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <el-button type="primary" @click="openCreate">新建员工</el-button>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="username" label="账号" width="140" />
      <el-table-column prop="nickname" label="姓名" width="140" />
      <el-table-column prop="role" label="角色" width="100">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">{{ row.role === 'admin' ? '管理员' : '员工' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status === 'active' ? '正常' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button text @click="openEdit(row)">编辑</el-button>
          <el-button text type="danger" :disabled="row.id === 1" @click="remove(row)">删除</el-button>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { userApi } from '@/api'

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
  await ElMessageBox.confirm(`确定删除员工 ${row.username}？`, '提示', { type: 'warning' })
  await userApi.remove(row.id); load()
}
onMounted(load)
</script>
