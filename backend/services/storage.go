package services

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SourceImageRelPath 返回原图在 UploadDir 下的相对路径：
//
//	products/{pid}/source/{filename}    （有 productID）
//	misc/source/{filename}                （无 productID 兜底，按当前 schema 不应出现）
//
// 同时作为 URL 的一部分（"/uploads/" + 此值）。
func SourceImageRelPath(productID *uint, filename string) string {
	if productID != nil {
		return filepath.Join("products", fmt.Sprintf("%d", *productID), "source", filename)
	}
	return filepath.Join("misc", "source", filename)
}

// GeneratedImageRelPath 生成图相对路径：products/{pid}/generated/...，无 pid 走 misc
func GeneratedImageRelPath(productID *uint, filename string) string {
	if productID != nil {
		return filepath.Join("products", fmt.Sprintf("%d", *productID), "generated", filename)
	}
	return filepath.Join("misc", "generated", filename)
}

// ResolveUploadPath 把相对路径拼成 server 端绝对路径
func ResolveUploadPath(uploadDir, relPath string) string {
	return filepath.Join(uploadDir, relPath)
}

// BuildImageURL 把 server 端绝对路径转成 /uploads/... 公开 URL。
// 反斜杠统一换成正斜杠，URL 段是正斜杠，Windows 部署也能跑。
// 反算失败时（文件不在 UploadDir 下）降级用 basename。
func BuildImageURL(uploadDir, fullPath string) string {
	rel, err := filepath.Rel(uploadDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "/uploads/" + filepath.Base(fullPath)
	}
	return "/uploads/" + filepath.ToSlash(rel)
}
