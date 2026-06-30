<template>
  <div>
    <el-row :gutter="16" class="mb-24">
      <el-col :span="6"><div class="page-card"><div class="text-muted">产品</div><div style="font-size:28px;font-weight:600">{{ stats.products }}</div></div></el-col>
      <el-col :span="6"><div class="page-card"><div class="text-muted">原图</div><div style="font-size:28px;font-weight:600">{{ stats.images }}</div></div></el-col>
      <el-col :span="6"><div class="page-card"><div class="text-muted">生成图</div><div style="font-size:28px;font-weight:600">{{ stats.gallery }}</div></div></el-col>
      <el-col :span="6"><div class="page-card"><div class="text-muted">卖点</div><div style="font-size:28px;font-weight:600">{{ stats.points }}</div></div></el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="page-card">
          <h3 style="margin-top:0">最近生成</h3>
          <el-table :data="latestGallery" stripe>
            <el-table-column label="预览" width="100">
              <template #default="{ row }">
                <img :src="row.url" style="width:80px;height:80px;object-fit:cover;border-radius:4px;" />
              </template>
            </el-table-column>
            <el-table-column prop="prompt" label="Prompt" show-overflow-tooltip />
            <el-table-column prop="modelName" label="模型" width="140" />
            <el-table-column prop="styleName" label="风格" width="100" />
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="page-card">
          <h3 style="margin-top:0">快速开始</h3>
          <el-steps direction="vertical" :active="0">
            <el-step title="上传产品原图" description="支持 JPG / PNG，自动解析产品" />
            <el-step title="AI 自动提取卖点与 Prompt" description="视觉模型识别并写入卖点表" />
            <el-step title="选择模型 + 风格预设生成" description="调用生图模型输出电商级图片" />
            <el-step title="保存到图库，筛选/下载" description="图库支持多维筛选与下载" />
          </el-steps>
          <el-button type="primary" size="large" style="margin-top:16px" @click="$router.push('/upload')">立即开始</el-button>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { productApi, imageApi, galleryApi, sellingPointApi } from '@/api'
import { formatDateTime } from '@/utils/format'

const stats = ref({ products: 0, images: 0, gallery: 0, points: 0 })
const latestGallery = ref([])

onMounted(async () => {
  const [ps, imgs, gs, sps] = await Promise.all([
    productApi.list(), imageApi.list(), galleryApi.list(), sellingPointApi.list()
  ])
  stats.value = { products: ps.length, images: imgs.length, gallery: gs.length, points: sps.length }
  latestGallery.value = gs.slice(0, 10)
})
</script>
