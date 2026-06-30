<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <div>
        <el-button type="primary" @click="openCreate">新建产品</el-button>
      </div>
      <div class="text-muted" style="font-size:12px">
        创建后到详情里逐张上传原图，每次上传会自动 AI 解析卖点 + prompt
      </div>
    </div>
    <el-table :data="list" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          <a class="name-link" @click.prevent="openDetail(row)">{{ row.name }}</a>
        </template>
      </el-table-column>
      <el-table-column label="原图" width="140">
        <template #default="{ row }">
          <div v-if="row.coverImageUrl" class="src-cell" @click="openDetail(row)" :title="`查看 ${row.sourceImageCount || 1} 张原图`">
            <img :src="row.coverImageUrl" class="src-thumb" />
            <el-tag v-if="(row.sourceImageCount || 0) > 1" size="small" type="info">+{{ (row.sourceImageCount || 1) - 1 }}</el-tag>
          </div>
          <span v-else class="text-muted" style="font-size:12px">未上传</span>
        </template>
      </el-table-column>
      <el-table-column label="生成图" width="120">
        <template #default="{ row }">
          <div v-if="row.galleryCount > 0" class="gen-cell" @click="openGallery(row)" :title="`查看 ${row.galleryCount} 张生成图`">
            <img v-if="row.previewGalleryUrl" :src="row.previewGalleryUrl" class="gen-thumb" />
            <div v-else class="gen-thumb gen-thumb-placeholder">{{ row.galleryCount }}</div>
            <el-tag size="small" type="success">{{ row.galleryCount }} 张</el-tag>
          </div>
          <span v-else class="text-muted" style="font-size:12px">—</span>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button text type="primary" @click="openDetail(row)">查看</el-button>
            <el-button text @click="openGallery(row)">图库</el-button>
            <el-button text @click="openEdit(row)">改名</el-button>
            <el-button text type="danger" @click="remove(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建产品 -->
    <el-dialog v-model="dlg" title="新建产品" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称" required>
          <el-input
            v-model="dlgName"
            placeholder="给产品起个名（公司内不允许重复）"
            maxlength="60"
            show-word-limit
            :status="createChecker.inputStatus()"
            @keyup.enter="submit"
          />
          <div v-if="createChecker.message.value" class="name-hint" :class="`is-${createChecker.state.value}`">
            {{ createChecker.message.value }}
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" :disabled="!createChecker.canSubmit()" @click="submit">创建</el-button>
      </template>
    </el-dialog>

    <!-- 改名 -->
    <el-dialog v-model="editDlg" title="修改产品名" width="500px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="产品名" required>
          <el-input
            v-model="editForm.name"
            placeholder="给产品起个新名"
            maxlength="60"
            show-word-limit
            @keyup.enter="submitEdit"
          />
        </el-form-item>
        <el-form-item v-if="editingId" label="产品 ID">
          <span class="text-muted" style="font-size:12px">#{{ editingId }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDlg = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!editForm.name.trim() || editSubmitting"
          :loading="editSubmitting"
          @click="submitEdit"
        >保存</el-button>
      </template>
    </el-dialog>

    <!-- 产品详情抽屉 -->
    <el-drawer v-model="detailDrawer" :title="detail?.name ? `产品详情 · ${detail.name}` : '产品详情'" size="640px" direction="rtl" destroy-on-close>
      <div v-if="detail" class="detail-body">
        <!-- 顶部摘要 -->
        <div class="detail-summary">
          <div class="summary-row">
            <span class="text-muted" style="font-size:12px">ID</span>
            <b>#{{ detail.id }}</b>
          </div>
          <div class="summary-row">
            <span class="text-muted" style="font-size:12px">创建时间</span>
            <span>{{ formatDateTime(detail.createdAt) }}</span>
          </div>
          <div class="summary-row">
            <span class="text-muted" style="font-size:12px">原图数</span>
            <b>{{ detail.sourceImages?.length || 0 }}</b>
          </div>
          <div class="summary-row">
            <span class="text-muted" style="font-size:12px">生成图数</span>
            <b>{{ detail.galleryCount || 0 }}</b>
          </div>
        </div>

        <el-divider><span class="section-title">原图（每张都有独立的卖点 + prompt）</span></el-divider>

        <!-- 原图卡片网格 + 上传卡片 -->
        <div class="src-grid">
          <div v-for="img in detail.sourceImages" :key="img.id" class="src-card">
            <a :href="img.url" target="_blank" class="src-card-img-wrap">
              <img :src="img.url" />
            </a>
            <div class="src-card-body">
              <div class="src-card-title">
                <el-tag size="small" :type="img.analyzed ? 'success' : 'info'">
                  {{ img.analyzed ? '已解析' : '未解析' }}
                </el-tag>
                <span class="text-muted" style="font-size:11px;margin-left:6px">#{{ img.id }}</span>
              </div>
              <div class="src-card-prompt" :title="img.prompt">{{ img.prompt || '（暂无 prompt）' }}</div>
              <div v-if="img.sellingPoints?.length" class="src-card-sp">
                <el-tag v-for="(sp, i) in img.sellingPoints" :key="i" size="small" type="info" effect="plain" style="margin:2px">
                  {{ sp }}
                </el-tag>
              </div>
              <div class="src-card-actions">
                <el-button text type="success" size="small" @click="gotoGenerate(img)">去生图</el-button>
                <el-button text type="primary" size="small" @click="copyPrompt(img)">复制 Prompt</el-button>
                <el-button text type="danger" size="small" @click="removeSourceImage(img)">删除</el-button>
              </div>
            </div>
          </div>

          <!-- 上传新原图卡片 -->
          <div class="src-card src-card-upload">
            <div class="upload-wrap">
              <el-upload
                ref="uploadRef"
                :auto-upload="false"
                :limit="1"
                :on-change="onFileChange"
                :on-exceed="onExceed"
                accept="image/*"
                :show-file-list="false"
              >
                <div class="upload-trigger">
                  <el-icon class="el-icon--upload"><upload-filled /></el-icon>
                  <div class="upload-text">{{ pickedFile ? '已选择文件，点击上传' : '点击或拖拽图片到此处' }}</div>
                  <div class="upload-tip" v-if="pickedFile">{{ pickedFile.name }} ({{ formatSize(pickedFile.size) }})</div>
                </div>
              </el-upload>
            </div>
            <div class="src-card-body">
              <el-form label-width="84px" size="small">
                <el-form-item label="视觉模型">
                  <el-select v-model="uploadForm.modelConfigId" placeholder="自动选择最新可用" clearable style="width:100%">
                    <el-option
                      v-for="m in visionModels"
                      :key="m.id"
                      :label="`${m.name} (${m.modelName})`"
                      :value="m.id"
                    />
                  </el-select>
                </el-form-item>
              </el-form>
              <el-button type="primary" :loading="uploading" :disabled="!pickedFile" size="small" style="width:100%;margin-top:8px" @click="uploadSourceImage">
                上传并解析
              </el-button>
            </div>
          </div>
        </div>

        <el-divider><span class="section-title">生成图（最近 {{ Math.min(detailGallery.length, 6) }} 张）</span></el-divider>
        <el-empty v-if="!detailGallery.length" description="还没生过图，去图库生成" :image-size="80" />
        <div v-else class="gen-grid">
          <a v-for="g in detailGallery.slice(0, 6)" :key="g.id" :href="g.url" target="_blank" class="gen-mini">
            <img :src="g.url" />
          </a>
        </div>

        <div class="detail-footer">
          <el-button @click="detailDrawer = false">关闭</el-button>
          <el-button type="primary" @click="gotoGenerate">去上传与生图</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { productApi, imageApi, galleryApi, modelConfigApi, sourceImageApi } from '@/api'
import { formatDateTime } from '@/utils/format'
import { confirmDelete } from '@/utils/confirmDelete'
import { useNameChecker } from '@/utils/useNameChecker'

const router = useRouter()
const list = ref([])
const dlg = ref(false)
// 弹窗里单独搞一个 dlgName，避免 form.name 嵌套对象触发 watch；
// 实时查重 + 必填 + 不重复都满足才允许「创建」。
const dlgName = ref('')
const createChecker = useNameChecker(
  dlgName,
  (name) => productApi.checkName(name).then(r => r.data || r),
  { debounceMs: 300 }
)

// 改名
const editDlg = ref(false)
const editForm = ref({ name: '' })
const editingId = ref(null)
const editSubmitting = ref(false)

// 详情抽屉
const detailDrawer = ref(false)
const detail = ref(null)
const detailGallery = ref([])
const visionModels = ref([])

// 上传新原图
const uploadRef = ref()
const pickedFile = ref(null)
const uploading = ref(false)
const uploadForm = ref({ modelConfigId: null })

const load = async () => { list.value = await productApi.list() }

const openGallery = (row) => {
  router.push({ name: 'Gallery', query: { productId: row.id } })
}
const gotoGallery = () => {
  if (!detail.value) return
  router.push({ name: 'Gallery', query: { productId: detail.value.id } })
}
const gotoGenerate = (img) => {
  if (!detail.value) return
  const query = { productId: detail.value.id }
  if (img) query.sourceImageId = img.id
  router.push({ name: 'Upload', query })
}

const openCreate = () => {
  dlgName.value = ''
  dlg.value = true
}
const submit = async () => {
  const name = dlgName.value.trim()
  if (!name) { ElMessage.warning('请输入产品名'); return }
  if (!createChecker.canSubmit()) {
    ElMessage.warning(createChecker.message.value || '产品名校验未通过')
    return
  }
  try {
    await productApi.create({ name })
    dlg.value = false
    ElMessage.success('已创建')
    load()
  } catch (e) {
    // 后端兜底（比如并发抢名）：错误提示在 http 拦截器已弹
  }
}

// 改名
const openEdit = (row) => {
  editingId.value = row.id
  editForm.value = { name: row.name || '' }
  editDlg.value = true
}
const submitEdit = async () => {
  if (!editingId.value) return
  const name = editForm.value.name.trim()
  if (!name) { ElMessage.warning('产品名不能为空'); return }
  editSubmitting.value = true
  try {
    await productApi.update(editingId.value, { name })
    editDlg.value = false
    ElMessage.success('已改名')
    load()
  } finally {
    editSubmitting.value = false
  }
}

const openDetail = async (row) => {
  detailDrawer.value = true
  detail.value = await productApi.get(row.id)
  // 同时拉一份该产品的 Gallery（只读缩略图）
  try {
    detailGallery.value = await galleryApi.list({ productId: row.id })
  } catch { detailGallery.value = [] }
}

const remove = async (row) => {
  try {
    await confirmDelete({
      label: '产品名',
      expected: row.name,
      title: `将永久删除产品「${row.name}」`,
      description: '其下原图与生成图保留但不再关联该产品（共享公司图库可继续查看）。',
    })
  } catch { return }
  await productApi.remove(row.id)
  ElMessage.success('已删除')
  load()
}

// 上传原图
const onFileChange = (file) => {
  pickedFile.value = file.raw
}
const onExceed = () => ElMessage.warning('一次只能上传一张图')
const formatSize = (n) => {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

const uploadSourceImage = async () => {
  if (!detail.value || !pickedFile.value) return
  uploading.value = true
  try {
    const r = await sourceImageApi.upload(detail.value.id, pickedFile.value, uploadForm.value.modelConfigId)
    ElMessage.success(`已上传并解析，新增 ${r.sellingPoints?.length || 0} 条卖点`)
    // 重新拉详情拿到最新 sourceImages
    detail.value = await productApi.get(detail.value.id)
    pickedFile.value = null
    if (uploadRef.value) uploadRef.value.clearFiles()
    // 列表里的原图数没有变化（这里没有专门暴露），等下次 load
  } finally { uploading.value = false }
}

const removeSourceImage = async (img) => {
  try {
    await confirmDelete({
      label: '原图 ID',
      expected: img.id,
      title: `将永久删除原图 #${img.id}`,
      description: '关联的 AI 卖点会一并保留，其他员工上传的原图不受影响。',
    })
  } catch { return }
  await imageApi.remove(img.id)
  ElMessage.success('已删除')
  detail.value = await productApi.get(detail.value.id)
}

const copyPrompt = async (img) => {
  if (!img.prompt) { ElMessage.warning('该图尚未解析'); return }
  try {
    await navigator.clipboard.writeText(img.prompt)
    ElMessage.success('Prompt 已复制')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

onMounted(async () => {
  await load()
  try {
    const ms = await modelConfigApi.list()
    visionModels.value = ms.filter(m => m.type === 'vision')
  } catch { visionModels.value = [] }
})
</script>

<style scoped>
.name-hint { font-size: 12px; line-height: 1.4; padding: 4px 0; }
.name-hint.is-ok        { color: var(--el-color-success); }
.name-hint.is-checking  { color: var(--el-color-warning); }
.name-hint.is-duplicate,
.name-hint.is-invalid   { color: var(--el-color-danger); }
.name-hint.is-error     { color: var(--el-color-danger); }

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
.name-link {
  color: var(--el-color-primary);
  cursor: pointer;
  font-weight: 500;
}
.name-link:hover { text-decoration: underline; }

.src-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
}
.src-cell:hover .src-thumb { transform: scale(1.05); }
.src-thumb {
  width: 44px; height: 44px; border-radius: 4px; overflow: hidden;
  background: #f5f5f5; border: 1px solid #ebeef5;
  object-fit: cover; display: block;
  transition: transform .15s ease;
}

.gen-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
}
.gen-cell:hover .gen-thumb { transform: scale(1.05); }
.gen-thumb {
  width: 36px;
  height: 36px;
  border-radius: 4px;
  object-fit: cover;
  background: #f5f5f5;
  display: block;
  border: 1px solid #ebeef5;
  transition: transform .15s ease;
}
.gen-thumb-placeholder {
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: #909399; font-weight: 600;
}

.detail-body { padding: 0 16px 16px; }
.detail-summary {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px;
  background: #fafafa; padding: 12px; border-radius: 6px;
}
.summary-row { display: flex; flex-direction: column; gap: 4px; }
.section-title { font-size: 13px; color: #909399; }

.src-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}
.src-card {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  overflow: hidden;
  background: #fff;
  display: flex;
  flex-direction: column;
}
.src-card-img-wrap {
  display: block;
  width: 100%;
  height: 140px;
  background: #f5f5f5;
  overflow: hidden;
}
.src-card-img-wrap img { width: 100%; height: 100%; object-fit: cover; display: block; }
.src-card-body { padding: 10px; flex: 1; display: flex; flex-direction: column; gap: 6px; }
.src-card-title { display: flex; align-items: center; }
.src-card-prompt {
  font-size: 12px; color: #606266;
  line-height: 1.4;
  max-height: 50px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.src-card-sp { display: flex; flex-wrap: wrap; min-height: 24px; }
.src-card-actions { display: flex; justify-content: space-between; margin-top: auto; padding-top: 4px; }

.src-card-upload { border-style: dashed; }
.upload-wrap { padding: 12px; }
.upload-trigger {
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  padding: 16px 8px;
  text-align: center;
  cursor: pointer;
  transition: border-color .2s;
}
.upload-trigger:hover { border-color: var(--el-color-primary); }
.upload-text { font-size: 12px; color: #606266; margin-top: 4px; }
.upload-tip { font-size: 11px; color: #909399; margin-top: 4px; word-break: break-all; }

.gen-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 6px;
}
.gen-mini {
  display: block;
  aspect-ratio: 1;
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid #ebeef5;
  background: #f5f5f5;
}
.gen-mini img { width: 100%; height: 100%; object-fit: cover; display: block; }

.detail-footer {
  display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px;
}
</style>
