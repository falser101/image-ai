<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <el-button type="primary" @click="openCreate">新建风格</el-button>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="160" />
      <el-table-column prop="description" label="说明" />
      <el-table-column prop="prompt" label="Prompt" show-overflow-tooltip />
      <el-table-column prop="negative" label="负向" show-overflow-tooltip />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button text @click="openEdit(row)">编辑</el-button>
          <el-button text type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="dlg" :title="form.id ? '编辑风格' : '新建风格'" width="560px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="form.description" /></el-form-item>
        <el-form-item label="Prompt"><el-input v-model="form.prompt" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="负向词"><el-input v-model="form.negative" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { styleApi } from '@/api'

const list = ref([])
const dlg = ref(false)
const form = ref({ id: null, name: '', description: '', prompt: '', negative: '' })
const load = async () => { list.value = await styleApi.list() }
const openCreate = () => { form.value = { id: null, name: '', description: '', prompt: '', negative: '' }; dlg.value = true }
const openEdit = (row) => { form.value = { ...row }; dlg.value = true }
const submit = async () => {
  if (!form.value.name || !form.value.prompt) { ElMessage.warning('请填写名称和Prompt'); return }
  if (form.value.id) await styleApi.update(form.value.id, form.value)
  else await styleApi.create(form.value)
  dlg.value = false; ElMessage.success('已保存'); load()
}
const remove = async (row) => {
  await ElMessageBox.confirm('确定删除该风格？', '提示', { type: 'warning' })
  await styleApi.remove(row.id); load()
}
onMounted(load)
</script>
