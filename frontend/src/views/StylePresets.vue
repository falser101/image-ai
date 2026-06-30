<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <div class="text-muted" style="font-size:12px">
        中文提示词：给运营/同事看的描述，方便在选风格时理解「这个风格是干什么的」。<br />
        英文提示词：实际发给生图模型的指令，逗号分隔的关键短语。
      </div>
      <el-button type="primary" @click="openCreate">新建风格</el-button>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column prop="description" label="说明" width="180" show-overflow-tooltip />
      <el-table-column prop="promptCN" label="中文提示词" min-width="180" show-overflow-tooltip />
      <el-table-column prop="promptEN" label="英文提示词" min-width="220" show-overflow-tooltip />
      <el-table-column prop="negative" label="负向" min-width="160" show-overflow-tooltip />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button text @click="openEdit(row)">编辑</el-button>
            <el-button text type="danger" @click="remove(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dlg" :title="form.id ? '编辑风格' : '新建风格'" width="640px">
      <el-form :model="form" label-width="96px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" maxlength="255" show-word-limit placeholder="一句话讲清楚这风格适合什么场景" />
        </el-form-item>
        <el-form-item label="中文提示词">
          <el-input
            v-model="form.promptCN"
            type="textarea"
            :rows="3"
            placeholder="例：白底柔光，主体居中，电商主图风格"
          />
          <div class="text-muted" style="font-size:12px;margin-top:2px">给人看的，选风格时一眼能懂。</div>
        </el-form-item>
        <el-form-item label="英文提示词" required>
          <el-input
            v-model="form.promptEN"
            type="textarea"
            :rows="3"
            placeholder="例：clean white background, soft studio lighting, product centered, commercial e-commerce style"
          />
          <div class="text-muted" style="font-size:12px;margin-top:2px">实际发给生图模型，逗号分隔关键短语。</div>
        </el-form-item>
        <el-form-item label="负向词">
          <el-input
            v-model="form.negative"
            type="textarea"
            :rows="2"
            placeholder="例：blurry, low quality, text, watermark"
          />
        </el-form-item>
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
import { ElMessage } from 'element-plus'
import { styleApi } from '@/api'
import { confirmDelete } from '@/utils/confirmDelete'

// 空表单：含全部字段（id 是编辑时才用）
const emptyForm = () => ({
  id: null,
  name: '',
  description: '',
  prompt: '',     // 兼容老字段：保存时回填 = promptEN
  promptCN: '',   // 中文提示词
  promptEN: '',   // 英文提示词（给模型）
  negative: ''
})

const list = ref([])
const dlg = ref(false)
const form = ref(emptyForm())

const load = async () => { list.value = await styleApi.list() }

const openCreate = () => { form.value = emptyForm(); dlg.value = true }

const openEdit = (row) => {
  // 编辑时把老 Prompt 兜底到 PromptEN 输入框（避免英文输入框空白）
  form.value = {
    ...emptyForm(),
    ...row,
    promptEN: row.promptEN || row.prompt || ''
  }
  dlg.value = true
}

const submit = async () => {
  if (!form.value.name || !form.value.name.trim()) { ElMessage.warning('请填写名称'); return }
  if (!form.value.promptEN || !form.value.promptEN.trim()) { ElMessage.warning('请填写英文提示词（给模型用）'); return }
  // 兼容老字段：新数据把 PromptEN 同步到 Prompt，保证后端 NOT NULL 列不空 + 老读取路径仍可用
  const payload = { ...form.value, prompt: form.value.promptEN }
  if (form.value.id) await styleApi.update(form.value.id, payload)
  else await styleApi.create(payload)
  dlg.value = false
  ElMessage.success('已保存')
  load()
}

const remove = async (row) => {
  try {
    await confirmDelete({
      label: '风格名',
      expected: row.name,
      title: `将永久删除风格「${row.name}」`,
      description: '历史上用此风格生成的图不受影响（它们的 Prompt 已合并落库）。',
    })
  } catch { return }
  await styleApi.remove(row.id)
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
