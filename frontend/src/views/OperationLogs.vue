<template>
  <div class="page-card">
    <div class="filter-bar mb-12">
      <el-select v-model="q.range" style="width:120px" @change="loadAll">
        <el-option label="今天" value="day" />
        <el-option label="本周" value="week" />
        <el-option label="本月" value="month" />
        <el-option label="全部" value="" />
      </el-select>
      <el-select v-model="q.userId" placeholder="所有人" clearable filterable style="width:160px" @change="loadAll">
        <el-option v-for="u in users" :key="u.id" :label="u.username" :value="u.id" />
      </el-select>
      <el-select v-model="q.action" placeholder="动作类型" clearable style="width:140px" @change="loadAll">
        <el-option label="POST" value="POST" />
        <el-option label="PUT" value="PUT" />
        <el-option label="DELETE" value="DELETE" />
        <el-option label="AI 解析" value="ai.analyze" />
        <el-option label="AI 生图" value="ai.generate" />
      </el-select>
      <el-input v-model="q.keyword" placeholder="搜索资源/ID/详情/账号" clearable style="width:240px" @keyup.enter="loadAll" @clear="loadAll" />
      <el-switch v-model="q.onlyAi" active-text="只看 AI 调用" @change="loadAll" />
      <el-button @click="loadAll">查询</el-button>
      <span class="text-muted" style="font-size:12px;margin-left:auto">
        {{ q.range === 'day' ? '今天' : q.range === 'week' ? '本周' : q.range === 'month' ? '本月' : '全部' }} ·
        AI 调用 {{ stats?.totalCalls || 0 }} 次 ·
        合计 {{ formatTokens(stats?.totalTokens || 0) }} tokens
      </span>
    </div>

    <!-- 按用户的 token 消耗汇总卡片 -->
    <div class="stats-row mb-12">
      <div class="stats-summary">
        <div class="stats-num">{{ formatTokens(stats?.totalTokens || 0) }}</div>
        <div class="stats-label">总 Token 消耗</div>
        <div class="stats-sub">
          输入 {{ formatTokens(stats?.totalPrompt || 0) }} · 输出 {{ formatTokens(stats?.totalCompletion || 0) }} · {{ stats?.totalCalls || 0 }} 次
        </div>
      </div>
      <div v-for="u in (stats?.users || [])" :key="u.UserID" class="user-card">
        <div class="uc-head">
          <span class="uc-name">{{ u.Username }}</span>
          <el-tag size="small" type="info">{{ u.CallCount }} 次</el-tag>
        </div>
        <div class="uc-main">
          <span class="uc-total">{{ formatTokens(u.Tokens) }}</span>
          <span class="uc-unit">tokens</span>
        </div>
        <div class="uc-sub">
          输入 {{ formatTokens(u.TokensPrompt) }} · 输出 {{ formatTokens(u.TokensCompletion) }}
        </div>
        <div class="uc-bar">
          <div class="uc-bar-fill" :style="{ width: barPct(u.Tokens) + '%' }"></div>
        </div>
      </div>
      <div v-if="!(stats?.users || []).length" class="stats-empty">
        {{ q.range || q.onlyAi ? '该条件下无 AI 记录' : '暂无 AI 调用记录' }}
      </div>
    </div>

    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column label="账号" width="140">
        <template #default="{ row }">
          <span :class="{'username-admin': row.role === 'admin'}">{{ row.username }}</span>
        </template>
      </el-table-column>
      <el-table-column label="动作" width="150">
        <template #default="{ row }">
          <el-tag :type="actionTagType(row.action)" size="small" effect="dark">
            {{ actionLabel(row.action) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="资源" min-width="220">
        <template #default="{ row }">
          <div class="resource-cell">
            <span class="res-name">{{ resourceName(row.resource) }}</span>
            <span v-if="row.resourceId" class="res-id">#{{ row.resourceId }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="详情" min-width="260" show-overflow-tooltip>
        <template #default="{ row }">{{ row.detail }}</template>
      </el-table-column>
      <el-table-column label="Token 消耗" width="170" align="right">
        <template #default="{ row }">
          <template v-if="row.tokens > 0">
            <div class="tokens-total">{{ formatTokens(row.tokens) }}</div>
            <div class="tokens-sub">
              输入 {{ formatTokens(row.tokensPrompt) }} · 输出 {{ formatTokens(row.tokensCompletion) }}
            </div>
          </template>
          <span v-else class="text-muted" style="font-size:12px">—</span>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column label="时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && !list.length" description="当前条件下没有日志" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { logApi, userApi } from '@/api'
import { formatDateTime } from '@/utils/format'

const list = ref([])
const users = ref([])
const stats = ref(null)
const loading = ref(false)
const q = ref({ range: '', userId: null, action: '', keyword: '', onlyAi: false })

const loadAll = async () => {
  loading.value = true
  try {
    const params = {
      range: q.value.range,
      userId: q.value.userId || '',
      action: q.value.action,
      keyword: q.value.keyword,
      onlyAi: q.value.onlyAi ? '1' : '',
    }
    const [ls, st] = await Promise.all([logApi.list(params), logApi.stats(params)])
    list.value = ls || []
    stats.value = st || null
  } finally {
    loading.value = false
  }
}

const formatTokens = (n) => {
  n = Number(n) || 0
  if (n < 1000) return String(n)
  if (n < 10000) return (n / 1000).toFixed(1) + 'k'
  if (n < 1_000_000) return Math.round(n / 1000) + 'k'
  return (n / 1_000_000).toFixed(2) + 'M'
}

const actionLabel = (a) => {
  if (!a) return ''
  if (a === 'ai.analyze') return 'AI 解析'
  if (a === 'ai.generate') return 'AI 生图'
  return a
}

const actionTagType = (a) => {
  if (a === 'POST') return 'primary'
  if (a === 'PUT') return 'warning'
  if (a === 'DELETE') return 'danger'
  if (a && a.startsWith('ai.')) return 'success'
  return 'info'
}

const resourceName = (r) => {
  if (!r) return ''
  // /api/products/:id → 产品；/api/users/:id → 员工；model → 模型
  if (r === 'model') return 'AI 模型'
  if (r.includes('/products')) return '产品'
  if (r.includes('/users')) return '员工'
  if (r.includes('/selling-points')) return '卖点'
  if (r.includes('/images')) return '原图'
  if (r.includes('/gallery')) return '生成图'
  if (r.includes('/ai/')) return 'AI 调用'
  if (r.includes('/model-configs')) return '模型配置'
  if (r.includes('/style-presets')) return '风格预设'
  if (r.includes('/auth')) return '账号'
  return r
}

const barPct = (n) => {
  const total = Number(stats.value?.totalTokens) || 0
  if (!total) return 0
  return Math.min(100, Math.round((Number(n) / total) * 100))
}

onMounted(async () => {
  try {
    users.value = await userApi.list()
  } catch { users.value = [] }
  loadAll()
})
</script>

<style scoped>
.filter-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.stats-row {
  display: flex;
  align-items: stretch;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px;
  background: linear-gradient(180deg, #f6faff 0%, #f0f5ff 100%);
  border: 1px solid #e6efff;
  border-radius: 6px;
}
.stats-summary {
  min-width: 200px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #4f7cff 0%, #6f5cff 100%);
  color: #fff;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.stats-num { font-size: 28px; font-weight: 700; line-height: 1.2; }
.stats-label { font-size: 12px; opacity: 0.9; }
.stats-sub { font-size: 11px; opacity: 0.85; }

.user-card {
  flex: 1 1 200px;
  min-width: 200px;
  max-width: 260px;
  padding: 10px 14px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.uc-head { display: flex; align-items: center; justify-content: space-between; }
.uc-name { font-size: 13px; color: var(--el-text-color-regular); font-weight: 500; }
.uc-main { display: flex; align-items: baseline; gap: 4px; }
.uc-total { font-size: 22px; font-weight: 700; color: var(--el-text-color-primary); }
.uc-unit { font-size: 11px; color: var(--el-text-color-secondary); }
.uc-sub { font-size: 11px; color: var(--el-text-color-secondary); }
.uc-bar {
  height: 4px; background: #eef1f8; border-radius: 2px; margin-top: 2px; overflow: hidden;
}
.uc-bar-fill {
  height: 100%; background: linear-gradient(90deg, #4f7cff, #6f5cff);
  transition: width .25s ease;
}
.stats-empty {
  flex: 1; min-width: 200px;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: var(--el-text-color-secondary);
}

.resource-cell {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}
.res-name { color: var(--el-text-color-regular); font-weight: 500; }
.res-id {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px; color: var(--el-color-primary);
  background: #f0f5ff; padding: 0 6px; border-radius: 3px;
}

.username-admin {
  color: var(--el-color-danger);
  font-weight: 600;
}

.tokens-total {
  font-weight: 700;
  color: var(--el-color-success);
}
.tokens-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}
</style>
