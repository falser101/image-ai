<template>
  <div class="page-card">
    <div class="flex items-center justify-between mb-12">
      <div class="flex items-center gap-12">
        <el-input v-model="q.keyword" placeholder="搜索 prompt/模型/风格" clearable style="width:240px" @keyup.enter="load" />
        <el-select v-model="q.productId" placeholder="产品" clearable filterable style="width:200px" @change="load">
          <el-option v-for="p in products" :key="p.id" :label="p.name" :value="p.id" />
        </el-select>
        <el-select v-model="q.modelConfigId" placeholder="模型" clearable style="width:200px" @change="load">
          <el-option v-for="m in models" :key="m.id" :label="m.name" :value="m.id" />
        </el-select>
        <el-select v-model="q.styleId" placeholder="风格" clearable style="width:160px" @change="load">
          <el-option v-for="s in styles" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button @click="load">查询</el-button>
      </div>
      <div class="text-muted" style="font-size:12px">
        去「上传与生图」选产品图 + 风格生图，新图会出现在这里
      </div>
    </div>
    <el-empty v-if="!loading && list.length===0" description="暂无图片" />
    <el-row :gutter="12" v-else>
      <el-col v-for="item in list" :key="item.id" :xs="12" :sm="8" :md="6" :lg="4" class="mb-12">
        <div class="card-item">
          <div class="card-img-wrap">
            <a :href="item.url" target="_blank">
              <img :src="item.url" class="card-img" />
            </a>
            <a
              v-if="item.sourceImageUrl"
              :href="item.sourceImageUrl"
              target="_blank"
              class="source-thumb"
              title="点击查看原图"
              @click.stop
            >
              <img :src="item.sourceImageUrl" />
              <span class="source-tag">原图</span>
            </a>
          </div>
          <div class="card-body">
            <div class="text-muted" style="font-size:12px;height:36px;overflow:hidden">{{ item.prompt }}</div>
            <div class="flex items-center justify-between" style="margin-top:6px">
              <div>
                <el-tag size="small" type="info" v-if="item.productName">{{ item.productName }}</el-tag>
                <el-tag size="small" v-if="item.modelName" style="margin-left:4px">{{ item.modelName }}</el-tag>
                <el-tag size="small" type="success" v-if="item.styleName" style="margin-left:4px">{{ item.styleName }}</el-tag>
              </div>
              <el-button text type="danger" @click="remove(item)">删除</el-button>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { galleryApi, modelConfigApi, styleApi, productApi } from '@/api'

const route = useRoute()
const list = ref([])
const loading = ref(false)
const models = ref([])
const styles = ref([])
const products = ref([])
const q = ref({ keyword: '', productId: null, modelConfigId: null, styleId: null })

const load = async () => {
  loading.value = true
  try {
    list.value = await galleryApi.list(q.value)
  } finally { loading.value = false }
}
const remove = async (item) => {
  await ElMessageBox.confirm('确定删除该图片？', '提示', { type: 'warning' })
  await galleryApi.remove(item.id)
  ElMessage.success('已删除')
  load()
}
onMounted(async () => {
  // 从 ?productId=… 跳转过来时，自动设上过滤
  const pid = parseInt(route.query.productId)
  if (!isNaN(pid)) q.value.productId = pid
  const [ms, ss, ps] = await Promise.all([
    modelConfigApi.list(),
    styleApi.list(),
    productApi.list(),
  ])
  models.value = ms
  styles.value = ss
  products.value = ps
  load()
})
</script>

<style scoped>
.card-item { background:#fff;border:1px solid #ebeef5;border-radius:6px;overflow:hidden; }
.card-img-wrap { position:relative; }
.card-img { width:100%; height:180px; object-fit:cover; display:block; background:#f5f5f5; }
/* 原图角标缩略图：浮在生成图左上角，点击单独打开原图，不触发主图链接 */
.source-thumb {
  position:absolute; top:6px; left:6px;
  width:48px; height:48px;
  border:2px solid #fff; border-radius:4px; overflow:hidden;
  box-shadow:0 1px 4px rgba(0,0,0,.25);
  background:#000;
  display:block; line-height:0;
}
.source-thumb img { width:100%; height:100%; object-fit:cover; display:block; }
.source-tag {
  position:absolute; left:0; right:0; bottom:0;
  background:rgba(0,0,0,.55); color:#fff;
  font-size:10px; line-height:14px; text-align:center;
  letter-spacing:.5px;
}
.card-body { padding: 8px 10px; }
</style>