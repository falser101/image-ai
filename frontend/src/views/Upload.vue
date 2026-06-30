<template>
  <div class="page-card upload-page">
    <!-- ============== Stage 1: Entry ============== -->
    <template v-if="stage === 'entry'">
      <div class="entry-header">
        <h2 style="margin:0">生成商品图</h2>
        <div class="text-muted" style="font-size:13px;margin-top:4px">
          三步搞定：上传原图 → AI 自动识别卖点与 Prompt → 选风格一键生成
        </div>
      </div>

      <el-row :gutter="16" class="entry-grid" align="top">
        <el-col :xs="24" :md="12">
          <div class="entry-card entry-card--primary">
            <div class="entry-card-title">📷 上传新商品图</div>

            <!-- 产品名输入：必填且不能重复 -->
            <div class="entry-card-name">
              <div class="entry-card-model-label">
                产品名 <span style="color:var(--el-color-danger)">*</span>
              </div>
              <el-input
                v-model="newProductName"
                :status="nameChecker.inputStatus()"
                placeholder="给商品起个名（公司内不允许重复）"
                maxlength="60"
                show-word-limit
                clearable
              />
              <div v-if="nameChecker.message.value" class="name-hint" :class="`is-${nameChecker.state.value}`">
                {{ nameChecker.message.value }}
              </div>
            </div>

            <el-upload
              drag
              :show-file-list="false"
              :http-request="handleNewUpload"
              accept="image/*"
              :before-upload="beforeUpload"
              :disabled="uploading || !canUploadNew"
              style="width:100%"
            >
              <el-icon class="el-icon--upload" style="font-size:48px;color:#909399"><upload-filled /></el-icon>
              <div class="el-upload__text">
                {{ uploading ? '上传与 AI 解析中…' : (canUploadNew ? '把商品原图拖到这里，或<em>点击上传</em>' : '请先在上方填写可用产品名') }}
              </div>
              <template #tip>
                <div class="text-muted" style="font-size:12px;line-height:1.6">
                  自动新建产品 · 自动识别卖点 · 自动生成 Prompt
                </div>
              </template>
            </el-upload>
            <div class="entry-card-model">
              <div class="entry-card-model-label">视觉模型（用于识别卖点与 Prompt）</div>
              <el-select
                v-model="uploadVisionModelId"
                placeholder="自动选择默认"
                clearable
                style="width:100%"
              >
                <el-option
                  v-for="m in visionModels"
                  :key="m.id"
                  :label="`${m.name} (${m.modelName})`"
                  :value="m.id"
                />
              </el-select>
              <div v-if="!visionModels.length" class="text-muted" style="font-size:12px;margin-top:4px">
                暂未配置视觉模型，将用 mock 解析（6 条固定卖点 + 默认 Prompt）
              </div>
            </div>
            <el-alert
              v-if="uploadError"
              :title="uploadError"
              type="error"
              show-icon
              :closable="false"
              style="margin-top:12px"
            />
          </div>
        </el-col>

        <el-col :xs="24" :md="12">
          <div class="entry-card entry-card--primary">
            <div class="entry-card-title">📦 已有产品上传新图</div>
            <template v-if="products.length">
              <el-select
                v-model="pickedProductId"
                placeholder="选择一个产品"
                filterable
                size="large"
                style="width:100%"
                @change="loadProduct"
              >
                <el-option v-for="p in products" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>

              <!-- 选中产品后给出两条路径：上传新原图 / 选已有原图 -->
              <div v-if="currentProduct" class="existing-product">
                <div class="existing-product-summary">
                  已选：<b>{{ currentProduct.name }}</b>
                  <span class="text-muted" style="font-weight:normal;margin-left:6px">
                    ({{ sourceImages.length || 0 }} 张原图)
                  </span>
                </div>

                <div class="existing-product-actions">
                  <!-- 路径 1：上传新原图到该产品，自动 AI 解析 -->
                  <el-upload
                    :show-file-list="false"
                    :http-request="handleExistingUpload"
                    accept="image/*"
                    :before-upload="beforeUpload"
                    :disabled="uploading"
                    class="existing-upload"
                  >
                    <el-button
                      type="primary"
                      :loading="uploading"
                      style="width:100%"
                    >📤 上传新原图到该产品</el-button>
                  </el-upload>

                  <!-- 路径 2：从已有原图里挑一张去生图 -->
                  <el-button
                    style="width:100%"
                    :disabled="!sourceImages.length"
                    @click="onPickFromExisting"
                  >
                    🖼 选已有原图生成 ({{ sourceImages.length || 0 }})
                  </el-button>
                </div>

                <div
                  v-if="!sourceImages.length"
                  class="text-muted"
                  style="font-size:12px;margin-top:10px;line-height:1.5"
                >
                  该产品还没有原图，请用上方按钮上传一张
                </div>
              </div>
              <div v-else class="text-muted" style="font-size:12px;margin-top:14px">
                选个产品，然后上传新原图或挑已有原图生成
              </div>
            </template>
            <el-empty v-else description="还没有任何产品" :image-size="80">
              <el-button type="primary" @click="$router.push('/products')">去新建产品</el-button>
            </el-empty>
          </div>
        </el-col>
      </el-row>
    </template>

    <!-- ============== Stage 2: Choose source image ============== -->
    <template v-else-if="stage === 'chooseImage'">
      <div class="flex items-center justify-between mb-12">
        <div>
          <h3 style="margin:0">选一张原图</h3>
          <div class="text-muted" style="font-size:12px">产品：{{ currentProduct?.name }}</div>
        </div>
        <el-button @click="goEntry">← 返回</el-button>
      </div>
      <div v-if="loadingSource" class="text-muted">加载中…</div>
      <div v-else-if="!sourceImages.length" class="text-muted">该产品还没有原图</div>
      <div v-else class="src-pick">
        <div
          v-for="img in sourceImages"
          :key="img.id"
          class="src-pick-card"
          :class="{ active: form.sourceImageId === img.id }"
          @click="pickSource(img)"
        >
          <img :src="img.url" />
          <div class="src-pick-prompt" :title="img.prompt">{{ img.prompt || '（暂无 prompt）' }}</div>
        </div>
      </div>
    </template>

    <!-- ============== Stage 3: Generate ============== -->
    <template v-else-if="stage === 'generate'">
      <div class="flex items-center justify-between mb-12">
        <h3 style="margin:0">选风格 → 生成</h3>
        <el-button text @click="goEntry">← 换个产品</el-button>
      </div>

      <el-alert
        v-if="!imageModels.length"
        title="暂无可用的生图模型，请联系管理员先在「模型配置」里添加"
        type="warning"
        show-icon
        :closable="false"
        style="margin-bottom:16px"
      />

      <!-- 已选源图 + 卖点 + Prompt -->
      <div class="gen-source">
        <div class="gen-source-img-wrap">
          <img :src="pickedImage.url" />
          <div class="text-muted" style="font-size:11px;text-align:center;margin-top:6px">
            原图 #{{ pickedImage.id }}
          </div>
        </div>
        <div class="gen-source-info">
          <div class="gen-source-row">
            <div class="gen-source-label">已识别卖点</div>
            <div class="gen-source-tags">
              <el-tag
                v-for="(sp, i) in (pickedImage.sellingPoints || [])"
                :key="i"
                size="small"
                effect="plain"
                style="margin:2px"
              >{{ sp }}</el-tag>
              <span v-if="!pickedImage.sellingPoints?.length" class="text-muted" style="font-size:12px">无</span>
            </div>
          </div>
          <div class="gen-source-row">
            <div class="gen-source-label">Prompt（可手动调整）</div>
            <el-input v-model="form.prompt" type="textarea" :rows="3" placeholder="可手动调整" />
          </div>
        </div>
      </div>

      <!-- 风格 -->
      <div class="gen-section">
        <h4 class="gen-section-title">风格（可选）</h4>
        <div class="style-pills">
          <div
            class="style-pill"
            :class="{ active: !form.styleId }"
            @click="form.styleId = null"
          >不使用</div>
          <div
            v-for="s in styles"
            :key="s.id"
            class="style-pill"
            :class="{ active: form.styleId === s.id }"
            :title="s.description"
            @click="form.styleId = s.id"
          >{{ s.name }}</div>
        </div>
      </div>

      <!-- 输出尺寸 -->
      <div class="gen-section">
        <h4 class="gen-section-title">输出尺寸</h4>
        <el-radio-group v-model="sizeKey">
          <el-radio-button v-for="s in sizeOptions" :key="s.key" :value="s.key">
            {{ s.short }}
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 高级设置，默认折叠 -->
      <el-collapse class="gen-advanced">
        <el-collapse-item title="高级设置（一般不用改）" name="adv">
          <el-form label-width="120px" size="default">
            <el-form-item label="生图模型">
              <el-select v-model="form.modelConfigId" style="width:300px">
                <el-option
                  v-for="m in imageModels"
                  :key="m.id"
                  :label="`${m.name} (${m.modelName})`"
                  :value="m.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="使用原图作参考">
              <el-switch v-model="form.useAsSubject" />
              <span class="text-muted" style="margin-left:8px;font-size:12px">
                仅 MiniMax 接口生效；非人像商品建议关闭
              </span>
            </el-form-item>
            <el-form-item label="Prompt 优化">
              <el-switch v-model="form.promptOptimizer" />
              <span class="text-muted" style="margin-left:8px;font-size:12px">
                让 MiniMax 自动改写 prompt 后再生成
              </span>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>

      <!-- 生成按钮 -->
      <div class="gen-actions">
        <el-button
          type="primary"
          size="large"
          :loading="generating"
          :disabled="!canGenerate"
          @click="doGenerate"
          style="min-width:160px"
        >{{ generating ? '生成中…' : '生成图片' }}</el-button>
        <el-button size="large" @click="goChooseImage" v-if="!generated">换个原图</el-button>
      </div>

      <!-- 结果 -->
      <div v-if="generated" class="gen-result">
        <el-divider><span class="text-muted">生成结果</span></el-divider>
        <div class="gen-result-card">
          <a :href="generated.imageUrl" target="_blank">
            <img :src="generated.imageUrl" />
          </a>
          <el-tag
            v-if="generated.status === 'fallback'"
            type="warning"
            effect="dark"
            style="position:absolute;top:8px;left:8px"
          >占位图</el-tag>
        </div>
        <div class="gen-result-actions">
          <el-button type="primary" @click="doGenerate" :loading="generating">同款再生成一张</el-button>
          <el-button @click="gotoGallery">查看该产品图库</el-button>
          <el-button @click="resetAll">完成</el-button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { aiApi, modelConfigApi, styleApi, productApi, sourceImageApi } from '@/api'
import { useNameChecker } from '@/utils/useNameChecker'

const route = useRoute()
const router = useRouter()

// ============== State ==============
const stage = ref('entry') // 'entry' | 'chooseImage' | 'generate'
const products = ref([])
const imageModels = ref([])
const visionModels = ref([])
const uploadVisionModelId = ref(null)
const newProductName = ref('') // 新建产品的自定义名（必填且不能重复）
// 实时查重：必填 + 唯一都满足才允许继续上传
const nameChecker = useNameChecker(
  newProductName,
  (name) => productApi.checkName(name).then(r => r.data || r),
  { debounceMs: 300 }
)
const styles = ref([])
const sourceImages = ref([])
const currentProduct = ref(null)
const pickedProductId = ref(null)
const loadingSource = ref(false)
const uploading = ref(false)
const uploadError = ref('')
const generating = ref(false)
const generated = ref(null)
const sizeKey = ref('1024x1024')

const form = ref({
  productId: null,
  sourceImageId: null,
  prompt: '',
  modelConfigId: null,
  styleId: null,
  useAsSubject: false,
  promptOptimizer: false,
  width: 1024,
  height: 1024,
})

const sizeOptions = [
  { key: '1024x1024', short: '1:1',   w: 1024, h: 1024 },
  { key: '1280x720',  short: '16:9',  w: 1280, h: 720 },
  { key: '1152x864',  short: '4:3',   w: 1152, h: 864 },
  { key: '1248x832',  short: '3:2',   w: 1248, h: 832 },
  { key: '832x1248',  short: '2:3',   w: 832,  h: 1248 },
  { key: '864x1152',  short: '3:4',   w: 864,  h: 1152 },
  { key: '720x1280',  short: '9:16',  w: 720,  h: 1280 },
  { key: '1344x576',  short: '21:9',  w: 1344, h: 576 },
]

const pickedImage = computed(() =>
  sourceImages.value.find(img => img.id === form.value.sourceImageId) || {}
)

const canGenerate = computed(() =>
  Boolean(form.value.productId && form.value.sourceImageId && form.value.modelConfigId)
)

// 「上传新商品图」按钮的可用条件：产品名必填 + 查重可用
const canUploadNew = computed(() => {
  if (!newProductName.value.trim()) return false
  return nameChecker.canSubmit()
})

// ============== Stage transitions ==============
const goEntry = () => {
  stage.value = 'entry'
  form.value.productId = null
  form.value.sourceImageId = null
  form.value.prompt = ''
  pickedProductId.value = null
  currentProduct.value = null
  sourceImages.value = []
  generated.value = null
}

const goChooseImage = () => {
  stage.value = 'chooseImage'
  form.value.sourceImageId = null
  form.value.prompt = ''
  generated.value = null
}

// loadProduct 把产品详情拉进 state，不做导航。stage 1 右卡用：
// 用户选中产品后只更新数据，由两个按钮决定下一步（上传 / 选已有）。
const loadProduct = async (id) => {
  if (!id) {
    currentProduct.value = null
    sourceImages.value = []
    return
  }
  loadingSource.value = true
  try {
    currentProduct.value = await productApi.get(id)
    sourceImages.value = currentProduct.value.sourceImages || []
  } catch (e) {
    ElMessage.error(e?.message || '加载产品失败')
    currentProduct.value = null
    sourceImages.value = []
  } finally {
    loadingSource.value = false
  }
}

// onPickFromExisting 把「选已有原图去生成」封装成一个按钮事件。
// 0 张：按钮 disabled；1 张：直接进生成；>=2：进 chooseImage 选图。
const onPickFromExisting = () => {
  if (!currentProduct.value) return
  if (sourceImages.value.length === 0) {
    ElMessage.warning('该产品还没有原图，请先上传一张')
    return
  }
  if (sourceImages.value.length === 1) {
    pickSource(sourceImages.value[0])
  } else {
    stage.value = 'chooseImage'
  }
}

const pickSource = (img) => {
  form.value.productId = currentProduct.value.id
  form.value.sourceImageId = img.id
  form.value.prompt = img.prompt || ''
  stage.value = 'generate'
}

// ============== Upload a fresh image (new product path) ==============
const beforeUpload = (file) => {
  if (file.size > 20 * 1024 * 1024) {
    ElMessage.warning('图片不能超过 20MB')
    return false
  }
  return true
}

const handleNewUpload = async ({ file }) => {
  uploading.value = true
  uploadError.value = ''
  try {
    // 1) 产品名必填且不能重复：用户输入优先；空 / 重复 / 校验未过都阻断上传
    const inputName = (newProductName.value || '').trim().slice(0, 60)
    if (!inputName) {
      ElMessage.warning('请先填写产品名')
      uploading.value = false
      return
    }
    if (!nameChecker.canSubmit()) {
      const msg = nameChecker.state.value === 'duplicate'
        ? `已有同名产品：「${inputName}」，换一个名字吧`
        : (nameChecker.message.value || '产品名校验未通过')
      ElMessage.warning(msg)
      uploading.value = false
      return
    }
    const name = inputName
    const product = await productApi.create({ name })

    // 2) 上传 + AI 解析（不指定 vision 模型时，后端按 fallback 跑 mock 或默认模型）
    const r = await sourceImageApi.upload(product.id, file, uploadVisionModelId.value || null)

    // 3) 把刚解析的图塞进当前上下文，跳到生成页
    const img = {
      id: r.image.id,
      url: r.image.url,
      prompt: r.prompt || '',
      sellingPoints: r.sellingPoints || [],
      analyzed: true,
    }
    currentProduct.value = { id: product.id, name: product.name }
    sourceImages.value = [img]
    form.value.productId = product.id
    form.value.sourceImageId = img.id
    form.value.prompt = r.prompt || ''
    stage.value = 'generate'
    newProductName.value = '' // 消费完清空，避免下次误用

    // 异步刷新侧栏/下拉里的产品列表（不阻塞当前流程）
    productApi.list().then(ps => { products.value = ps }).catch(() => {})
  } catch (e) {
    uploadError.value = e?.message || '上传或解析失败，请重试'
  } finally {
    uploading.value = false
  }
}

// ============== Upload new source image to an existing product ==============
// 复用同一套后端接口 sourceImageApi.upload(productId, file, modelConfigId)。
// 成功解析后直接跳到 generate 阶段，把这张新图作为源图。
const handleExistingUpload = async ({ file }) => {
  if (!currentProduct.value) {
    ElMessage.warning('请先选择产品')
    return
  }
  uploading.value = true
  uploadError.value = ''
  try {
    const r = await sourceImageApi.upload(
      currentProduct.value.id,
      file,
      uploadVisionModelId.value || null
    )
    // 把新图塞进当前 sourceImages 列表，让用户后续能在 generate 页看到
    const img = {
      id: r.image.id,
      url: r.image.url,
      prompt: r.prompt || '',
      sellingPoints: r.sellingPoints || [],
      analyzed: true,
    }
    sourceImages.value = [...sourceImages.value, img]
    form.value.productId = currentProduct.value.id
    form.value.sourceImageId = img.id
    form.value.prompt = r.prompt || ''
    stage.value = 'generate'
    ElMessage.success(`已上传并解析，新增 ${r.sellingPoints?.length || 0} 条卖点`)
  } catch (e) {
    uploadError.value = e?.message || '上传或解析失败，请重试'
  } finally {
    uploading.value = false
  }
}

// ============== Generate ==============
const onSizeChange = (k) => {
  const opt = sizeOptions.find(s => s.key === k)
  if (opt) { form.value.width = opt.w; form.value.height = opt.h }
}
watch(sizeKey, onSizeChange)

const doGenerate = async () => {
  if (!canGenerate.value) {
    ElMessage.warning('请补齐：产品 / 原图 / 生图模型')
    return
  }
  onSizeChange(sizeKey.value)
  generating.value = true
  try {
    const payload = {
      productId: form.value.productId,
      sourceImageId: form.value.sourceImageId,
      prompt: form.value.prompt,
      modelConfigId: form.value.modelConfigId,
      styleId: form.value.styleId || undefined,
      useAsSubject: form.value.useAsSubject,
      promptOptimizer: form.value.promptOptimizer,
      width: form.value.width,
      height: form.value.height,
    }
    // 当前默认实现：useAsSubject 关掉时不传 sourceImageId 给生图后端
    if (!payload.useAsSubject) payload.sourceImageId = null
    generated.value = await aiApi.generate(payload)
    ElMessage.success('生成成功')
  } finally {
    generating.value = false
  }
}

const gotoGallery = () => {
  router.push({ name: 'Gallery', query: { productId: form.value.productId } })
}

const resetAll = () => {
  generated.value = null
  goEntry()
}

onMounted(async () => {
  const [ms, ss, ps] = await Promise.all([
    modelConfigApi.list(),
    styleApi.list(),
    productApi.list(),
  ])
  imageModels.value = ms.filter(m => m.type === 'image' || m.type === '')
  visionModels.value = ms.filter(m => m.type === 'vision')
  uploadVisionModelId.value = visionModels.value[0]?.id || null
  styles.value = ss
  products.value = ps
  form.value.modelConfigId = imageModels.value[0]?.id || null

  // 从 ?productId=… 跳转过来时，自动选中产品并加载它的原图
  const pid = parseInt(route.query.productId)
  if (!isNaN(pid) && products.value.some(p => p.id === pid)) {
    await loadProduct(pid)
    // 如果 URL 还指定了具体原图，直接选中它（多图产品也能一键直达）
    const sid = parseInt(route.query.sourceImageId)
    if (!isNaN(sid)) {
      const target = sourceImages.value.find(img => img.id === sid)
      if (target) {
        pickSource(target)
      } else {
        onPickFromExisting()
      }
    } else {
      onPickFromExisting()
    }
  }
})
</script>

<style scoped>
/* 让整页占满屏幕高度，两张入口卡片等高分摊 */
.upload-page {
  min-height: calc(100vh - 110px);
  display: flex;
  flex-direction: column;
}
.entry-header { margin-bottom: 16px; }
.entry-grid { margin-top: 16px; flex: 1; }
.entry-card {
  background: #fafafa;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  flex-direction: column;
}
.entry-card--primary {
  height: 100%;
  min-height: 360px;
}
.entry-card-title { font-size: 15px; font-weight: 600; margin-bottom: 12px; }
.entry-card-name { margin-bottom: 12px; }
.entry-card-model { margin-top: 16px; }
.entry-card-model-label { font-size: 12px; color: #909399; margin-bottom: 4px; }

/* 产品名校验状态文案 */
.name-hint { font-size: 12px; margin-top: 4px; line-height: 1.4; }
.name-hint.is-ok        { color: var(--el-color-success); }
.name-hint.is-checking  { color: var(--el-color-warning); }
.name-hint.is-duplicate,
.name-hint.is-invalid   { color: var(--el-color-danger); }
.name-hint.is-error     { color: var(--el-color-danger); }

/* 已有产品区：选中后给出两条路径 */
.existing-product { margin-top: 16px; display: flex; flex-direction: column; }
.existing-product-summary {
  font-size: 13px; margin-bottom: 12px;
  padding: 8px 10px; background: #f0f5ff; border-radius: 4px;
  border: 1px dashed #c0d4f5;
}
.existing-product-actions {
  display: flex; flex-direction: column; gap: 8px;
}
/* el-upload 默认会渲染一个透明的 trigger 包装层；这里让它不抢布局 */
.existing-upload { display: block; }
.existing-upload .el-upload { display: block; }

.src-pick {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 10px;
}
.src-pick-card {
  position: relative;
  border: 2px solid #ebeef5;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color .15s ease, transform .15s ease;
  background: #fff;
}
.src-pick-card:hover { border-color: #c0c4cc; transform: translateY(-1px); }
.src-pick-card.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary);
}
.src-pick-card img {
  width: 100%; height: 100px; object-fit: cover; display: block; background: #f5f5f5;
}
.src-pick-prompt {
  font-size: 11px; color: #606266; padding: 6px 8px; height: 36px; line-height: 1.4;
  overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
}

.gen-source {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  margin-bottom: 20px;
}
.gen-source-img-wrap img {
  width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: 4px;
  border: 1px solid #ebeef5; background: #f5f5f5;
}
.gen-source-info { display: flex; flex-direction: column; gap: 12px; }
.gen-source-label { font-size: 12px; color: #909399; margin-bottom: 4px; }
.gen-source-tags { display: flex; flex-wrap: wrap; }

.gen-section { margin-bottom: 20px; }
.gen-section-title { font-size: 14px; font-weight: 600; margin: 0 0 8px; }

.style-pills {
  display: flex; flex-wrap: wrap; gap: 8px;
}
.style-pill {
  padding: 6px 14px;
  border: 1px solid #dcdfe6;
  border-radius: 20px;
  cursor: pointer;
  font-size: 13px;
  transition: all .15s ease;
  background: #fff;
  user-select: none;
}
.style-pill:hover { border-color: var(--el-color-primary); color: var(--el-color-primary); }
.style-pill.active {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #fff;
}

.gen-advanced { margin-bottom: 20px; }
.gen-actions { margin-bottom: 24px; }

.gen-result { margin-top: 24px; }
.gen-result-card {
  position: relative;
  display: inline-block;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #ebeef5;
}
.gen-result-card img {
  display: block;
  max-width: 480px;
  border-radius: 6px;
}
.gen-result-actions { margin-top: 16px; display: flex; gap: 8px; flex-wrap: wrap; }
</style>
