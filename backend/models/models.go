package models

import "time"

// User 用户表
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string         `gorm:"size:128;not null" json:"-"`
	Nickname  string         `gorm:"size:64" json:"nickname"`
	Role      string         `gorm:"size:16;default:employee" json:"role"` // admin | employee
	Status    string         `gorm:"size:16;default:active" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt *time.Time     `gorm:"index" json:"-"`
}

// Product 产品
type Product struct {
	ID         uint         `gorm:"primaryKey" json:"id"`
	UserID     uint         `gorm:"index;not null" json:"userId"`
	Name       string       `gorm:"size:128" json:"name"`
	ImageID    *uint        `gorm:"index" json:"imageId"`
	Prompt     string       `gorm:"type:text" json:"prompt"`
	SellingPts string       `gorm:"type:text" json:"sellingPoints"` // JSON 数组
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
	DeletedAt  *time.Time   `gorm:"index" json:"-"`
}

// SellingPoint 卖点
type SellingPoint struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index;not null" json:"userId"`
	ProductID *uint      `gorm:"index" json:"productId"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Source    string     `gorm:"size:16" json:"source"` // manual | ai
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// Image 上传的原图
type Image struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"userId"`
	ProductID  *uint      `gorm:"index" json:"productId"`
	Filename   string     `gorm:"size:255;not null" json:"filename"`
	Path       string     `gorm:"size:512;not null" json:"path"`
	Size       int64      `json:"size"`
	MimeType   string     `gorm:"size:64" json:"mimeType"`
	Width      int        `json:"width"`
	Height     int        `json:"height"`
	Prompt     string     `gorm:"type:text" json:"prompt"`
	SellingPts string     `gorm:"type:text" json:"sellingPoints"` // JSON 数组
	Analyzed   bool       `gorm:"default:false" json:"analyzed"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	DeletedAt  *time.Time `gorm:"index" json:"-"`
}

// Gallery 生成结果图
type Gallery struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"userId"`
	ProductID     *uint      `gorm:"index" json:"productId"`
	SourceImageID *uint      `gorm:"index" json:"sourceImageId"`
	Filename      string     `gorm:"size:255;not null" json:"filename"`
	Path          string     `gorm:"size:512;not null" json:"path"`
	URL           string     `gorm:"size:1024" json:"url"`
	Prompt        string     `gorm:"type:text" json:"prompt"`
	ModelConfigID *uint      `gorm:"index" json:"modelConfigId"`
	ModelName     string     `gorm:"size:64" json:"modelName"`
	StyleID       *uint      `gorm:"index" json:"styleId"`
	StyleName     string     `gorm:"size:64" json:"styleName"`
	Size          int64      `json:"size"`
	Width         int        `json:"width"`
	Height        int        `json:"height"`
	Status        string     `gorm:"size:16;default:success" json:"status"` // success | failed
	Error         string     `gorm:"type:text" json:"error"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `gorm:"index" json:"-"`
}

// StylePreset 风格预设
type StylePreset struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:64;not null" json:"name"`
	Description string     `gorm:"size:255" json:"description"`
	Prompt      string     `gorm:"type:text;not null" json:"prompt"`
	Negative    string     `gorm:"type:text" json:"negative"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:64;not null" json:"name"`
	Provider   string     `gorm:"size:32" json:"provider"` // openai | minimax | custom ...
	Type       string     `gorm:"size:16" json:"type"`     // vision | image | text
	BaseURL    string     `gorm:"size:255" json:"baseUrl"`
	APIKey     string     `gorm:"size:255" json:"apiKey"`
	ModelName  string     `gorm:"size:64" json:"modelName"`
	Extra      string     `gorm:"type:text" json:"extra"` // JSON
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	DeletedAt  *time.Time `gorm:"index" json:"-"`
}

// OssConfig OSS存储配置（单例）
type OssConfig struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Provider   string    `gorm:"size:32" json:"provider"` // local | aliyun | tencent | aws
	Endpoint   string    `gorm:"size:255" json:"endpoint"`
	Bucket     string    `gorm:"size:64" json:"bucket"`
	AccessKey  string    `gorm:"size:255" json:"accessKey"`
	SecretKey  string    `gorm:"size:255" json:"secretKey"`
	Region     string    `gorm:"size:32" json:"region"`
	Prefix     string    `gorm:"size:128" json:"prefix"`
	PublicHost string    `gorm:"size:255" json:"publicHost"`
	Enabled    bool      `gorm:"default:false" json:"enabled"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// OperationLog 操作日志
type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	Username  string    `gorm:"size:64" json:"username"`
	Action    string    `gorm:"size:64" json:"action"`
	Resource  string    `gorm:"size:64" json:"resource"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

// AITask AI任务（用于轮询）
type AITask struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	UserID     uint      `gorm:"index" json:"userId"`
	Type       string    `gorm:"size:16" json:"type"` // analyze | generate
	Status     string    `gorm:"size:16" json:"status"`
	Progress   int       `json:"progress"`
	Result     string    `gorm:"type:text" json:"result"`
	Error      string    `gorm:"type:text" json:"error"`
	Payload    string    `gorm:"type:text" json:"payload"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
