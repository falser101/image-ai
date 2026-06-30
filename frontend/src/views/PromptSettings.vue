<template>
  <div class="page-card">
    <div class="header mb-16">
      <div>
        <h3 style="margin:0">提示词配置</h3>
        <div class="text-muted" style="font-size:12px;margin-top:4px;line-height:1.6">
          「上传与生图」第一步里，AI 视觉模型会用这里配的 system 模板从产品原图中提取
          <b>产品名 + 卖点 + 生图 Prompt</b>。改完保存立即生效，不需要重启服务。
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="load" :loading="loading">重新读取</el-button>
        <el-button @click="onReset">恢复默认</el-button>
        <el-button type="primary" :loading="saving" :disabled="!canSave" @click="onSave">保存</el-button>
      </div>
    </div>

    <div class="meta mb-12">
      <span class="text-muted" style="font-size:12px">
        单行单例（id=1），最近更新：
        <b>{{ updatedAtText }}</b>
      </span>
      <span class="text-muted" style="font-size:12px;margin-left:16px">
        字节 {{ content.length }} · 行数 {{ lineCount }}
      </span>
    </div>

    <el-input
      v-model="content"
      type="textarea"
      :rows="20"
      spellcheck="false"
      :autosize="{ minRows: 18, maxRows: 40 }"
      :placeholder="placeholder"
      style="font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 13px;"
    />

    <div v-if="lastError" class="mt-12">
      <el-alert :title="lastError" type="error" show-icon :closable="false" />
    </div>
    <div v-if="lastWarning" class="mt-12">
      <el-alert :title="lastWarning" type="warning" show-icon :closable="false" />
    </div>

    <div class="tips mt-16">
      <div class="tips-title">编辑提示</div>
      <ul>
        <li><b>JSON 输出 schema</b>：模型会严格按 <code>sellingPoints / prompt</code>（外加 productName）输出，schema 变了模型可能返字段缺失导致分析失败</li>
        <li><b>风格 / 要素</b>：默认要求 ① 主体 ② 光线 ③ 构图 ④ 背景 ⑤ 画面风格。要新增要素就在 prompt 段加一条编号</li>
        <li><b>字数</b>：每条卖点建议 10–25 字；prompt 80–150 字。极端长（&gt;4k token）会让响应变慢且更易截断</li>
        <li><b>生效时机</b>：保存后下一次 Analyze 调用就生效。正在跑的任务不受影响</li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateTime } from '@/utils/format'
import { promptSettingsApi } from '@/api'

const content = ref('')
const initial = ref('')
const updatedAt = ref('')
const saving = ref(false)
const loading = ref(false)
const lastError = ref('')
const lastWarning = ref('')

const updatedAtText = computed(() => formatDateTime(updatedAt.value) || '未修改（当前为系统默认）')
const lineCount = computed(() => {
  const n = content.value.split('\n').length
  return n
})
const placeholder = '在这里粘贴 / 编辑 system prompt…'
const canSave = computed(() => content.value.trim() !== '' && content.value !== initial.value)

const load = async () => {
  loading.value = true
  try {
    const res = await promptSettingsApi.get()
    const d = res.data || res
    content.value = d.systemInstruction || ''
    initial.value = content.value
    updatedAt.value = d.updatedAt || ''
  } catch (e) {
    lastError.value = e?.message || '读取失败'
  } finally {
    loading.value = false
  }
}

const onSave = async () => {
  const text = content.value.trim()
  if (!text) {
    ElMessage.warning('提示词不能为空')
    return
  }
  saving.value = true
  try {
    const r = await promptSettingsApi.update({ systemInstruction: content.value })
    const d = r.data || r
    content.value = d.systemInstruction || content.value
    initial.value = content.value
    updatedAt.value = d.updatedAt || ''
    ElMessage.success('已保存')
  } catch (e) {
    lastError.value = e?.message || '保存失败'
  } finally {
    saving.value = false
  }
}

const onReset = async () => {
  try {
    await ElMessageBox.confirm('恢复为系统默认提示词？此操作会覆盖当前修改。', '提示', { type: 'warning' })
  } catch { return }
  saving.value = true
  try {
    const r = await promptSettingsApi.reset()
    const d = r.data || r
    content.value = d.systemInstruction || content.value
    initial.value = content.value
    ElMessage.success('已恢复默认')
    await load()
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.header-actions { display: flex; gap: 8px; flex-shrink: 0; }

.meta { display: flex; gap: 4px; align-items: center; }

.tips {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 14px 18px;
  background: #fafafa;
}
.tips-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.tips ul { margin: 0; padding-left: 18px; color: var(--el-text-color-regular); font-size: 12px; line-height: 1.9; }
.tips code {
  background: #f0f1f4; padding: 1px 6px; border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}
</style>
