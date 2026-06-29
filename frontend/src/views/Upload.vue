<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <h3 style="margin:0">快捷生图</h3>
      <div class="text-muted" style="font-size:12px">
        从已有产品图里选一张 → 选风格 → 直接生成，不做 AI 解析
      </div>
    </div>

    <el-form label-width="92px">
      <el-form-item label="选择产品" required>
        <el-select
          v-model="form.productId"
          placeholder="先选一个产品"
          filterable clearable
          style="width:100%;max-width:420px"
          @change="onProductChange"
        >
          <el-option v-for="p in products" :key="p.id" :label="p.name" :value="p.id" />
        </el-select>
        <div class="text-muted" style="font-size:12px;margin-top:4px">
          需要先去「产品」页给该产品上传原图
        </div>
      </el-form-item>

      <el-form-item v-if="form.productId" label="选择原图" required>
        <div v-if="loadingSource" class="text-muted">加载中…</div>
        <div v-else-if="sourceImages.length === 0" class="text-muted">
          该产品还没有原图
        </div>
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
            <el-icon v-if="form.sourceImageId === img.id" class="src-pick-check"><CircleCheckFilled /></el-icon>
          </div>
        </div>
      </el-form-item>

      <el-form-item label="生图 Prompt">
        <el-input v-model="form.prompt" type="textarea" :rows="3" placeholder="选择原图后会自动填入该图的 prompt，可继续编辑" />
      </el-form-item>

      <el-form-item label="生图模型" required>
        <el-select v-model="form.modelConfigId" placeholder="选择模型" style="width:340px" clearable>
          <el-option v-for="m in imageModels" :key="m.id" :label="m.name + ' (' + m.modelName + ')'" :value="m.id" />
        </el-select>
      </el-form-item>

      <el-form-item label="风格预设">
        <el-select v-model="form.styleId" placeholder="选择风格（可选）" style="width:300px" clearable>
          <el-option v-for="s in styles" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
      </el-form-item>

      <el-form-item label="输出尺寸">
        <el-select v-model="sizeKey" style="width:260px" @change="onSizeChange">
          <el-option v-for="s in sizeOptions" :key="s.key" :label="s.label" :value="s.key" />
        </el-select>
        <div class="text-muted" style="font-size:12px;margin-top:4px">
          后端会自动按比例映射到 provider 支持的 aspect_ratio（1:1 / 4:3 / 3:4 / 16:9 / 9:16 / 21:9 / 3:2 / 2:3）
        </div>
      </el-form-item>

      <el-form-item label="参考原图">
        <el-checkbox v-model="form.useAsSubject">
          使用本张原图作为角色参考 (subject_reference / character)
        </el-checkbox>
        <div class="text-muted" style="font-size:12px;margin-top:4px">
          ⚠️ 仅 MiniMax 接口生效，且仅支持 type=character。非人像商品图不要勾选。
        </div>
      </el-form-item>

      <el-form-item label="Prompt 优化">
        <el-switch v-model="form.promptOptimizer" />
        <span class="text-muted" style="margin-left:8px;font-size:12px">
          开启后 MiniMax 会对 prompt 自动改写再生成
        </span>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="generating" :disabled="!canGenerate" @click="doGenerate">
          生成图片
        </el-button>
      </el-form-item>
    </el-form>

    <el-divider />
    <div v-if="generated">
      <el-result icon="success" title="生成成功">
        <template #extra>
          <a :href="generated.imageUrl" target="_blank">
            <img :src="generated.imageUrl" style="max-width:320px;border-radius:6px;border:1px solid #eee" />
          </a>
          <div style="margin-top:12px">
            <el-button @click="gotoGallery">查看该产品的图库</el-button>
            <el-button @click="resetForm">再做一张</el-button>
          </div>
        </template>
      </el-result>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { CircleCheckFilled } from '@element-plus/icons-vue'
import { aiApi, modelConfigApi, styleApi, productApi } from '@/api'

const route = useRoute()
const router = useRouter()
const products = ref([])
const imageModels = ref([])
const styles = ref([])
const loadingSource = ref(false)
const generating = ref(false)
const sourceImages = ref([])
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
  { key: '1024x1024', label: '1024 × 1024  (1:1 方形)', w: 1024, h: 1024 },
  { key: '1280x720',  label: '1280 × 720   (16:9 横屏)', w: 1280, h: 720 },
  { key: '1152x864',  label: '1152 × 864   (4:3 横屏)', w: 1152, h: 864 },
  { key: '1248x832',  label: '1248 × 832   (3:2 横屏)', w: 1248, h: 832 },
  { key: '832x1248',  label: '832  × 1248  (2:3 竖屏)', w: 832,  h: 1248 },
  { key: '864x1152',  label: '864  × 1152  (3:4 竖屏)', w: 864,  h: 1152 },
  { key: '720x1280',  label: '720  × 1280  (9:16 竖屏)', w: 720,  h: 1280 },
  { key: '1344x576',  label: '1344 × 576   (21:9 超宽，仅 image-01)', w: 1344, h: 576 },
]

const canGenerate = computed(() => {
  return form.value.productId && form.value.sourceImageId && form.value.modelConfigId
})

const onProductChange = async (productId) => {
  form.value.sourceImageId = null
  form.value.prompt = ''
  sourceImages.value = []
  if (!productId) return
  loadingSource.value = true
  try {
    const detail = await productApi.get(productId)
    sourceImages.value = detail.sourceImages || []
    if (sourceImages.value.length === 1) pickSource(sourceImages.value[0])
  } finally {
    loadingSource.value = false
  }
}

const pickSource = (img) => {
  form.value.sourceImageId = img.id
  form.value.prompt = img.prompt || ''
}

const onSizeChange = (k) => {
  const opt = sizeOptions.find(s => s.key === k)
  if (opt) { form.value.width = opt.w; form.value.height = opt.h }
}

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
    if (!payload.useAsSubject) payload.sourceImageId = null
    generated.value = await aiApi.generate(payload)
    ElMessage.success('生成成功')
  } finally { generating.value = false }
}

const gotoGallery = () => {
  router.push({ name: 'Gallery', query: { productId: form.value.productId } })
}

const resetForm = () => {
  generated.value = null
  form.value.prompt = ''
}

onMounted(async () => {
  const [ms, ss, ps] = await Promise.all([
    modelConfigApi.list(),
    styleApi.list(),
    productApi.list(),
  ])
  imageModels.value = ms.filter(m => m.type === 'image' || m.type === '')
  styles.value = ss
  products.value = ps
  form.value.modelConfigId = imageModels.value[0]?.id || null

  // 从 ?productId=… 跳转过来时，自动选中产品并加载它的原图
  const pid = parseInt(route.query.productId)
  if (!isNaN(pid) && products.value.some(p => p.id === pid)) {
    await onProductChange(pid)
  }
})
</script>

<style scoped>
.src-pick {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
  width: 100%;
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
  width: 100%; height: 90px; object-fit: cover; display: block; background: #f5f5f5;
}
.src-pick-prompt {
  font-size: 11px;
  color: #606266;
  padding: 6px 8px;
  height: 44px;
  line-height: 1.4;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.src-pick-check {
  position: absolute;
  top: 6px;
  right: 6px;
  font-size: 22px;
  color: var(--el-color-primary);
  background: #fff;
  border-radius: 50%;
}
</style>