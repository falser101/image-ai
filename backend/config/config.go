package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port         string
	DBPath       string
	UploadDir    string
	JWTSecret    string
	DefaultAdmin struct {
		Username string
		Password string
	}
}

func Load() *Config {
	c := &Config{
		Port:      getenv("PORT", "8080"),
		DBPath:    getenv("DB_PATH", "./data.db"),
		UploadDir: getenv("UPLOAD_DIR", "./uploads"),
		JWTSecret: getenv("JWT_SECRET", "image-ai-default-secret-change-me"),
	}
	c.DefaultAdmin.Username = getenv("ADMIN_USERNAME", "admin")
	c.DefaultAdmin.Password = getenv("ADMIN_PASSWORD", "admin123")
	_ = os.MkdirAll(c.UploadDir, 0o755)
	_ = os.MkdirAll(filepath.Dir(c.DBPath), 0o755)
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
