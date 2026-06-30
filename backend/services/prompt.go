package services

import (
	"errors"

	"github.com/image-ai/backend/models"

	"gorm.io/gorm"
)

// DefaultSystemInstruction 是 AI 视觉模型识别产品图所用 system prompt 的默认值。
// 首次启动时没有 DB 行，落这条作为 seed。改的时候在「系统管理 → 提示词配置」编辑即可。
const DefaultSystemInstruction = `你是一名产品图分析助理 + AI 生图提示词工程师。系统会把用户登记好的产品名（hint）随图一起发给你，你的首要任务是判断「图与名是否一致」并如实输出。请严格按以下 JSON 输出，**不要**使用 markdown 代码块、不要任何解释文字。

输出 schema：
{
  "productName": "产品中文名（2-8字，不含品牌）",
  "sellingPoints": ["卖点1", "卖点2", "卖点3", "卖点4", "卖点5"],
  "prompt": "中文生图 prompt（80-150 字）"
}

字段要求：
- productName：简洁中文产品名，如「蓝牙耳机」「陶瓷茶壶」「户外帐篷」，**不要带品牌名**。
  · 当用户给的产品名 hint 与图明显一致（外观/材质/用途都吻合）时，直接沿用该名称。
  · 当 hint 与图明显冲突（例如 hint=「蓝牙耳机」但图是陶瓷茶壶），按图真实内容给出一个准确的品类名，不要照搬 hint。
  · 当用户没给 hint 时，按图识别。
- sellingPoints：4-6 条，每条 10-25 字，从外观 / 材质 / 工艺 / 功能 / 使用场景角度提炼，面向 C 端消费者。卖点要体现该真实品类的特性，而不是按 hint 强行虚构。
- prompt：中文生图 prompt，结构清晰、用逗号分隔，**不要用引号或代码块包裹**。必须包含以下五要素：
  ① 主体：产品外观、颜色、材质、关键造型细节；
  ② 光线：柔光 / 侧光 / 逆光 / 轮廓光 / 自然光 等具体光线类型；
  ③ 构图与机位：俯拍 / 45 度 / 平视 / 产品居中 / 三分构图 等；
  ④ 背景与场景：纯色背景（颜色）/ 生活场景（简述）/ 棚拍 等；
  ⑤ 画面风格：极简产品摄影 / 商业广告 / 电影感 / 国潮 / 杂志风 等。`

// GetSystemInstruction 从 DB 读当前管理员设置的 system prompt。
// 没找到时回退到 DB 里的 seed（首次启动自动写入），最后再回退到 DefaultSystemInstruction。
// 不做内存缓存：管理员改完立即生效。
func GetSystemInstruction(db *gorm.DB) (string, error) {
	if db == nil {
		return DefaultSystemInstruction, nil
	}
	var s models.PromptSettings
	err := db.First(&s, 1).Error
	if err == nil && s.SystemInstruction != "" {
		return s.SystemInstruction, nil
	}
	// 行不存在（理论上启动时已 seed，但 schema 极老/被人手动清表时可能没有），
	// 此时回写到 DB 并用默认值。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		seed := models.PromptSettings{ID: 1, SystemInstruction: DefaultSystemInstruction}
		if createErr := db.Create(&seed).Error; createErr == nil {
			return DefaultSystemInstruction, nil
		}
	}
	// 失败兜底：返回默认值，至少不挂掉业务
	return DefaultSystemInstruction, nil
}

// EnsureDefaultPromptSettings 在启动时 seed id=1 一行；幂等。
func EnsureDefaultPromptSettings(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var s models.PromptSettings
	if err := db.First(&s, 1).Error; err == nil {
		// 已存在就不覆盖，避免覆盖管理员修改过的值
		return nil
	}
	seed := models.PromptSettings{ID: 1, SystemInstruction: DefaultSystemInstruction}
	return db.Create(&seed).Error
}
