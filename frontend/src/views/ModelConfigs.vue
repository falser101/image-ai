<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <el-button type="primary" @click="openCreate">添加模型</el-button>
      <span class="text-muted">内置 provider 一键接入：选类型 → 填 API Key → 选模型</span>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" width="200" />
      <el-table-column prop="type" label="类型" width="90">
        <template #default="{ row }">
          <el-tag :type="row.type === 'vision' ? 'warning' : 'success'">
            {{ row.type === 'vision' ? '视觉' : '生图' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Provider" width="170">
        <template #default="{ row }">
          <el-tag size="small">{{ providerLabel(row.provider) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="modelName" label="模型" width="180" />
      <el-table-column label="BaseURL" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="text-muted" style="font-size:12px">{{ row.baseUrl || '—' }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="apiKey" label="API Key" width="120">
        <template #default="{ row }">
          <span class="text-muted" style="font-size:12px">
            {{ row.apiKey ? '已配置（不可见）' : '未配置' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="启用" width="80">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="(v) => toggle(row, v)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button text type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button text type="danger" @click="remove(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer
      v-model="dlg"
      direction="rtl"
      size="560px"
      :with-header="true"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">{{ form.id ? '编辑模型' : '添加模型' }}</span>
          <span class="text-muted" style="font-size:12px">Provider → API Key → 选模型</span>
        </div>
      </template>

      <el-steps :active="step" finish-status="success" simple class="mb-16">
        <el-step title="Provider / 类型" />
        <el-step title="API Key" />
      </el-steps>

      <el-form :model="form" label-width="100px">
        <el-form-item label="Provider">
          <el-select v-model="form.provider" filterable style="width:100%" @change="onProviderChange">
            <el-option v-for="p in providers" :key="p.key" :label="p.label" :value="p.key">
              <span style="float:left">{{ p.label }}</span>
              <el-tag v-if="p.builtIn" size="small" type="success" style="float:right;margin-left:8px">内置</el-tag>
            </el-option>
          </el-select>
          <div class="text-muted" style="font-size:12px;margin-top:4px">
            {{ currentProvider?.description }}
          </div>
        </el-form-item>

        <el-form-item label="类型">
          <el-radio-group v-model="form.type" @change="onTypeChange">
            <el-radio-button label="image">生图</el-radio-button>
            <el-radio-button label="vision">视觉</el-radio-button>
          </el-radio-group>
          <div class="text-muted" style="font-size:12px;margin-top:4px">
            生图调用 /v1/image_generation；视觉调用 /v1/chat/completions（多模态，支持图片/视频）
          </div>
        </el-form-item>

        <el-form-item label="API Key">
          <el-input v-model="form.apiKey" show-password :placeholder="apiKeyPlaceholder" />
          <div class="text-muted" style="font-size:12px;margin-top:4px">
            {{ apiKeyHint }}
          </div>
        </el-form-item>

        <el-form-item label="模型">
          <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
            <el-select
              v-model="form.modelName"
              filterable
              allow-create
              :loading="loadingModels"
              :no-data-text="modelNoDataText"
              :placeholder="modelPlaceholder"
              :style="form.type === 'image' ? 'flex:1;min-width:220px' : 'flex:1;min-width:200px'"
            >
              <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
            </el-select>
            <el-button
              v-if="form.type === 'vision'"
              :loading="fetchingModels"
              :disabled="!canFetch"
              @click="openFetchPicker"
            >
              获取模型列表
            </el-button>
          </div>
          <div class="text-muted" style="font-size:12px;margin-top:4px">
            {{ modelHint }}
          </div>
        </el-form-item>

        <template v-if="!currentProvider?.builtIn">
          <el-form-item label="BaseURL">
            <el-input v-model="form.baseUrl" placeholder="https://your-api.com/v1" />
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model="form.name" placeholder="给这条配置起个名" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="地址">
            <el-input :model-value="form.baseUrl" readonly />
            <div class="text-muted" style="font-size:12px;margin-top:4px">内置 provider 使用官方地址，不可修改</div>
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model="form.name" />
            <div class="text-muted" style="font-size:12px;margin-top:4px">已按 provider + 模型自动命名，可按需改</div>
          </el-form-item>
        </template>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <div style="display:flex;justify-content:flex-end;gap:8px">
          <el-button @click="dlg = false">取消</el-button>
          <el-button type="primary" :disabled="!canSave" @click="submit">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <el-dialog
      v-model="pickerVisible"
      :title="`从 ${currentProvider?.label || ''} 拉取的模型`"
      width="480px"
      @open="onPickerOpen"
    >
      <div class="text-muted" style="font-size:12px;margin-bottom:8px">
        勾选要加入下拉的模型（已选 {{ pickerSelected.length }} 个）
      </div>
      <el-input v-model="pickerFilter" placeholder="筛选" clearable size="small" style="margin-bottom:8px" />
      <el-checkbox-group v-model="pickerSelected" class="picker-list">
        <el-checkbox v-for="m in filteredPicker" :key="m" :value="m" class="picker-item">
          {{ m }}
        </el-checkbox>
      </el-checkbox-group>
      <div v-if="filteredPicker.length === 0" class="text-muted" style="padding:12px 0;text-align:center">
        无匹配模型
      </div>
      <template #footer>
        <el-button @click="pickerVisible = false">取消</el-button>
        <el-button type="primary" :disabled="pickerSelected.length === 0" @click="applyFetched">
          加入并使用第一个
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-card :deep(.el-table .row-actions) {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0;
  white-space: nowrap;
}
.page-card :deep(.el-table .row-actions .el-button) {
  padding: 0 6px;
}
.picker-list {
  max-height: 360px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 8px 12px;
}
.picker-item {
  display: flex;
  width: 100%;
  margin-right: 0;
  padding: 4px 0;
}
.drawer-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.drawer-title {
  font-size: 18px;
  font-weight: 600;
}
/* 抽屉内容区可滚动，表单项自然撑开 */
.page-card :deep(.el-drawer__body) {
  padding: 0 24px;
  overflow-y: auto;
}
.page-card :deep(.el-drawer__footer) {
  padding: 12px 24px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>

<script setup>
import { onMounted, ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { modelConfigApi } from '@/api'
import http from '@/api/http'
import { confirmDelete } from '@/utils/confirmDelete'

const list = ref([])
const dlg = ref(false)
const providers = ref([])
const loadingModels = ref(false)
const form = ref({ id: null, name: '', type: 'image', provider: 'minimax', baseUrl: '', apiKey: '', modelName: '', enabled: true })

// 弹窗内「临时」追加的模型列表（来自 /providers/fetch-models，按当前 type 区分）
const fetchedImage = ref([])
const fetchedVision = ref([])
// picker 弹窗
const pickerVisible = ref(false)
const fetchingModels = ref(false)
const pickerAll = ref([])        // 后端返回的全量（按当前 type 过滤后）
const pickerSelected = ref([])   // 用户多选结果
const pickerFilter = ref('')     // picker 内的筛选关键词

const currentProvider = computed(() => providers.value.find(p => p.key === form.value.provider))
const isBuiltIn = computed(() => !!currentProvider.value?.builtIn)
// 生图组写死 list（MiniMax 官方公布的两种 image 模型）；视觉组不预设，由「获取模型列表」拉到。
const HARDCODED_IMAGE_MODELS = ['image-01', 'image-01-live']
// 当前 type 的可选模型：
//   生图 + 内置 minimax → 写死 [image-01, image-01-live]（允许手输覆盖）
//   视觉 + 内置 minimax → 只有本次拉到的（fetchedVision）
//   自定义 provider → 都不预置，用户手输或拉
const modelOptions = computed(() => {
  const p = currentProvider.value
  if (!p) return []
  if (form.value.type === 'image') {
    if (isBuiltIn.value && p.key === 'minimax') {
      return HARDCODED_IMAGE_MODELS
    }
    return [...fetchedImage.value]
  }
  // type === 'vision'
  if (isBuiltIn.value) {
    return [...(p.visionModels || []), ...fetchedVision.value]
  }
  return [...fetchedVision.value]
})
// 「获取」按钮的可用条件：
//   1) provider 选了
//   2) 内置 provider：apiKey 可空（后端会用 DB 现存 key），custom 必须填
const canFetch = computed(() => {
  if (!form.value.provider) return false
  if (isBuiltIn.value) return true
  return !!form.value.apiKey && !!form.value.baseUrl
})
const modelPlaceholder = computed(() => {
  if (form.value.type === 'image') {
    return '选 image-01 / image-01-live，或输入自定义'
  }
  return fetchedVision.value.length === 0
    ? '点「获取模型列表」拉取，或手输'
    : '选一个模型，或输入自定义'
})
const modelNoDataText = computed(() => {
  if (form.value.type === 'image') return '可手输自定义模型名'
  return '点击「获取模型列表」按钮，或直接在输入框里手输'
})
const modelHint = computed(() => {
  if (form.value.type === 'image') return '生图模型：下拉只有 image-01 / image-01-live 两种；手输可填自定义模型名'
  return '视觉模型（多模态，支持图片/视频）。点击「获取模型列表」可拉取 MiniMax 当前可用的全部模型'
})
// picker 内的筛选结果
const filteredPicker = computed(() => {
  const q = pickerFilter.value.trim().toLowerCase()
  if (!q) return pickerAll.value
  return pickerAll.value.filter(m => m.toLowerCase().includes(q))
})
const autoName = computed(() => {
  const p = currentProvider.value
  if (!p) return ''
  const tag = form.value.modelName || (form.value.type === 'image' ? '生图' : '视觉')
  return `${p.label} ${tag}`.trim()
})
const apiKeyPlaceholder = computed(() => {
  if (form.value.id) return '留空表示不修改现有 Key'
  if (form.value.provider === 'minimax') return 'MiniMax 控制台申请的 API Key'
  if (form.value.provider === 'custom') return 'Bearer Token'
  return 'Bearer Token'
})
const apiKeyHint = computed(() => {
  if (form.value.id) return '编辑模式下输入框为空，已保存的 Key 不会显示也不会被覆盖；只有填写新值才会更新'
  if (form.value.provider === 'minimax') return '在 minimaxi.com 控制台 → API Keys 创建；不会明文回显'
  return '只保存在本地数据库；不会明文回显'
})
const canSave = computed(() => form.value.provider && form.value.modelName && form.value.type)
const step = computed(() => {
  if (!form.value.provider || !form.value.type) return 0
  // 编辑模式下 apiKey 可空，所以 step 直接到选模型
  if (form.value.id) return 1
  if (!form.value.apiKey) return 1
  return 1
})

const providerLabel = (key) => providers.value.find(p => p.key === key)?.label || key

const load = async () => { list.value = await modelConfigApi.list() }
const loadProviders = async () => {
  loadingModels.value = true
  try {
    providers.value = await http.get('/providers')
  } finally {
    loadingModels.value = false
  }
}

const onProviderChange = () => {
  if (currentProvider.value) {
    if (!form.value.baseUrl) form.value.baseUrl = currentProvider.value.baseUrl
    // 生图 + 内置 minimax 时，如果 modelName 不在写死列表里、且为空 → 默认选第一个
    if (isBuiltIn.value && currentProvider.value.key === 'minimax' && form.value.type === 'image') {
      if (!form.value.modelName && HARDCODED_IMAGE_MODELS.length > 0) {
        form.value.modelName = HARDCODED_IMAGE_MODELS[0]
      }
    }
  }
  // 切换 provider 时清空已拉取的临时列表
  fetchedImage.value = []
  fetchedVision.value = []
  refreshAutoName()
}
const onTypeChange = () => {
  // 切类型时 modelName 可能不匹配新组的预置，先清空
  form.value.modelName = ''
  onProviderChange()
}

const refreshAutoName = () => {
  if (!form.value.id && autoName.value) {
    form.value.name = autoName.value
  }
}
watch(() => [form.value.provider, form.value.modelName, form.value.type], refreshAutoName)

const openCreate = () => {
  form.value = {
    id: null, name: '', type: 'image', provider: 'minimax',
    baseUrl: 'https://api.minimaxi.com', apiKey: '', modelName: '', enabled: true
  }
  dlg.value = true
}
const openEdit = (row) => {
  // 编辑模式：永远不回显 apiKey（即使后端意外返回了原文也不显示）
  form.value = {
    id: row.id,
    name: row.name,
    type: row.type,
    provider: row.provider,
    baseUrl: row.baseUrl,
    apiKey: '',          // 强制清空：用户必须输入新值才会被更新
    modelName: row.modelName,
    enabled: row.enabled,
  }
  // 切换 provider/类型时清掉上次拉取的临时列表
  fetchedImage.value = []
  fetchedVision.value = []
  dlg.value = true
}

// 「获取模型列表」：先用表单当前值调后端拿到 all，再按当前 type 过滤进 picker
const openFetchPicker = async () => {
  pickerVisible.value = true
  fetchingModels.value = true
  pickerAll.value = []
  pickerSelected.value = []
  pickerFilter.value = ''
  try {
    const data = await http.post('/providers/fetch-models', {
      provider: form.value.provider,
      baseUrl: form.value.baseUrl,
      apiKey: form.value.apiKey,
      type: '',  // 拉全量，前端自己按 type 分
    })
    pickerAll.value = (data.all || []).slice()
    // 预选视觉组启发式匹配的几个（minimax-/abab 开头），方便用户一眼看到能用的
    pickerSelected.value = pickerAll.value.filter(m => {
      const lower = m.toLowerCase()
      return lower.startsWith('minimax-') || lower.startsWith('abab')
    })
  } catch (e) {
    // 错误消息在 http 拦截器里 ElMessage 了，这里关掉弹窗即可
    pickerVisible.value = false
  } finally {
    fetchingModels.value = false
  }
}
const onPickerOpen = () => {
  // 打开后聚焦筛选框（如果需要可后续加 ref）
}
// 把 picker 选中的模型追加到表单的「视觉」临时列表（按钮只在视觉组出现，所以只走 vision 分支）
const applyFetched = () => {
  if (pickerSelected.value.length === 0) return
  fetchedVision.value = [...new Set([...fetchedVision.value, ...pickerSelected.value])]
  form.value.modelName = pickerSelected.value[0]
  refreshAutoName()
  pickerVisible.value = false
  ElMessage.success(`已加入 ${pickerSelected.value.length} 个模型到下拉`)
}
const submit = async () => {
  if (!form.value.modelName) { ElMessage.warning('请选择或输入模型名'); return }
  if (!form.value.name) form.value.name = autoName.value
  const data = { ...form.value }
  if (data.id) {
    // 编辑模式下若 apiKey 留空，从 payload 里删掉这个字段，避免覆盖数据库里的现有 key
    if (!data.apiKey) delete data.apiKey
    await modelConfigApi.update(data.id, data)
  } else {
    if (!data.apiKey) { ElMessage.warning('新建模型必须填写 API Key'); return }
    await modelConfigApi.create(data)
  }
  dlg.value = false
  ElMessage.success('已保存')
  load()
}
const toggle = async (row, v) => {
  await modelConfigApi.update(row.id, { enabled: v })
  ElMessage.success('已更新'); load()
}
const remove = async (row) => {
  try {
    await confirmDelete({
      label: '模型名',
      expected: row.name,
      title: `将永久删除模型配置「${row.name}」`,
      description: '之后「上传与生图」「产品原图 AI 解析」将无法再选该模型。',
    })
  } catch { return }
  await modelConfigApi.remove(row.id)
  ElMessage.success('已删除')
  load()
}
onMounted(async () => {
  await loadProviders()
  await load()
})
</script>
