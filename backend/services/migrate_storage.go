package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"

	"gorm.io/gorm"
)

// MigrateStorageLayout 把老的扁平 uploads/* 文件挪到新的 products/{pid}/{source,generated}/ 布局。
// 启动时调用，幂等：已在新位置的会被跳过；缺失文件不阻塞、log warn。
// 同时更新 Image.Path / Gallery.Path / Gallery.URL 字段以保持和实际文件一致。
func MigrateStorageLayout(db *gorm.DB, cfg *config.Config) error {
	log.Printf("[migrate] storage layout migration starting, uploadDir=%s", cfg.UploadDir)

	var images []models.Image
	if err := db.Find(&images).Error; err != nil {
		return fmt.Errorf("load images: %w", err)
	}
	imgMoved, imgSkip, imgMissing := 0, 0, 0
	for i := range images {
		img := &images[i]
		newPath, ok := relocateFile(cfg.UploadDir, img.Path, SourceImageRelPath(img.ProductID, filepath.Base(img.Path)))
		if !ok {
			imgMissing++
			continue
		}
		if newPath == img.Path {
			imgSkip++
			continue
		}
		if err := db.Model(img).Update("path", newPath).Error; err != nil {
			log.Printf("[migrate] image id=%d update path failed: %v", img.ID, err)
			continue
		}
		imgMoved++
	}

	var galleries []models.Gallery
	if err := db.Find(&galleries).Error; err != nil {
		return fmt.Errorf("load galleries: %w", err)
	}
	galMoved, galSkip, galMissing := 0, 0, 0
	for i := range galleries {
		g := &galleries[i]
		newPath, ok := relocateFile(cfg.UploadDir, g.Path, GeneratedImageRelPath(g.ProductID, filepath.Base(g.Path)))
		if !ok {
			galMissing++
			continue
		}
		newURL := BuildImageURL(cfg.UploadDir, newPath)
		if newPath == g.Path && newURL == g.URL {
			galSkip++
			continue
		}
		updates := map[string]any{}
		if newPath != g.Path {
			updates["path"] = newPath
		}
		if newURL != g.URL {
			updates["url"] = newURL
		}
		if len(updates) > 0 {
			if err := db.Model(g).Updates(updates).Error; err != nil {
				log.Printf("[migrate] gallery id=%d update failed: %v", g.ID, err)
				continue
			}
		}
		galMoved++
	}

	log.Printf("[migrate] done: images moved=%d skip=%d missing=%d; galleries moved=%d skip=%d missing=%d",
		imgMoved, imgSkip, imgMissing, galMoved, galSkip, galMissing)
	return nil
}

// relocateFile 期望新相对路径在 UploadDir 下，挪文件过去。
// 返回 (newFullPath, true) = 成功（newFullPath 等于 oldPath 表示未移动）
// 返回 (oldPath, false)  = 文件不存在
func relocateFile(uploadDir, oldPath, newRelPath string) (string, bool) {
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return oldPath, false
		}
		log.Printf("[migrate] stat %s failed: %v", oldPath, err)
		return oldPath, false
	}
	newPath := filepath.Join(uploadDir, newRelPath)
	if newPath == oldPath {
		return newPath, true
	}
	// 新位置的子目录可能不存在
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		log.Printf("[migrate] mkdir %s failed: %v", filepath.Dir(newPath), err)
		return oldPath, true
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		log.Printf("[migrate] rename %s -> %s failed: %v", oldPath, newPath, err)
		return oldPath, true
	}
	return newPath, true
}
