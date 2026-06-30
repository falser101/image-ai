package handlers

import (
	"strconv"
	"time"

	"github.com/image-ai/backend/config"
	"github.com/image-ai/backend/models"
	"github.com/image-ai/backend/services"
	"github.com/image-ai/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewUserHandler(db *gorm.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{db: db, cfg: cfg}
}

func (h *UserHandler) List(c *gin.Context) {
	var list []models.User
	h.db.Order("id desc").Find(&list)
	utils.OK(c, list)
}

type userCreateReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req userCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, 400, "参数错误")
		return
	}
	var count int64
	h.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.Fail(c, 1003, "账号已存在")
		return
	}
	u, err := services.EnsureUser(h.db, req.Username, req.Password, req.Nickname, req.Role)
	if err != nil {
		utils.Fail(c, 500, "创建失败")
		return
	}
	utils.OK(c, u)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var u models.User
	if err := h.db.First(&u, id).Error; err != nil {
		utils.Fail(c, 404, "不存在")
		return
	}
	var patch map[string]any
	c.ShouldBindJSON(&patch)
	// 防止密码明文落入
	if _, ok := patch["password"]; ok {
		delete(patch, "password")
	}
	if pwd, ok := patch["newPassword"].(string); ok && pwd != "" {
		hash, _ := utils.HashPassword(pwd)
		patch["password"] = hash
		delete(patch, "newPassword")
	}
	if r, ok := patch["role"].(string); ok && r != "" && r != "admin" && r != "employee" {
		delete(patch, "role")
	}
	h.db.Model(&u).Updates(patch)
	utils.OK(c, u)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 1 {
		utils.Fail(c, 1004, "默认管理员不可删除")
		return
	}
	h.db.Delete(&models.User{}, id)
	utils.OK(c, nil)
}

type OperationLogHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewOperationLogHandler(db *gorm.DB, cfg *config.Config) *OperationLogHandler {
	return &OperationLogHandler{db: db, cfg: cfg}
}

func (h *OperationLogHandler) List(c *gin.Context) {
	q := h.db.Model(&models.OperationLog{})
	if uid := c.Query("userId"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	// 仅 AI 调用（默认 behavior: action 以 ai. 开头的；或者显式 tokenOnly=1）
	if c.Query("onlyAi") == "1" {
		q = q.Where("action LIKE ? OR action = ?", "ai.%", "ai.%")
	}
	if res := c.Query("resource"); res != "" {
		q = q.Where("resource = ?", res)
	}
	if kw := c.Query("keyword"); kw != "" {
		q = q.Where("resource LIKE ? OR resource_id LIKE ? OR detail LIKE ? OR username LIKE ?",
			"%"+kw+"%", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	// 时间范围：range=day|week|month（默认不过滤；前端可传）
	if start, end, ok := parseTimeRange(c.Query("range"), c.Query("startDate"), c.Query("endDate")); ok {
		q = q.Where("created_at BETWEEN ? AND ?", start, end)
	}
	var list []models.OperationLog
	q.Order("id desc").Limit(500).Find(&list)
	utils.OK(c, list)
}

// Stats 聚合 token 消耗。
//   range=day  → 今天 00:00 ~ 明天 00:00
//   range=week → 本周一 00:00 ~ 下周一 00:00（按本周第一天）
//   range=month→ 本月 1 号 00:00 ~ 下月 1 号 00:00
//   默认       → 全部
// 同时支持显式 startDate / endDate（YYYY-MM-DD），覆盖 range。
//
// 返回按 userId 聚合的 token 总数 + 调用次数 + 范围端点；并附 total 总数。
func (h *OperationLogHandler) Stats(c *gin.Context) {
	start, end, ok := parseTimeRange(c.Query("range"), c.Query("startDate"), c.Query("endDate"))
	q := h.db.Model(&models.OperationLog{}).Where("tokens > 0")
	if ok {
		q = q.Where("created_at BETWEEN ? AND ?", start, end)
	}
	if uid := c.Query("userId"); uid != "" {
		q = q.Where("user_id = ?", uid)
	}
	var rows []struct {
		UserID           uint
		Username         string
		Tokens           int
		TokensPrompt     int
		TokensCompletion int
		CallCount        int
	}
	q.Select("user_id, username, SUM(tokens) as tokens, SUM(tokens_prompt) as tokens_prompt, SUM(tokens_completion) as tokens_completion, COUNT(*) as call_count").
		Group("user_id, username").
		Order("tokens desc").
		Scan(&rows)
	if rows == nil {
		rows = []struct {
			UserID           uint
			Username         string
			Tokens           int
			TokensPrompt     int
			TokensCompletion int
			CallCount        int
		}{}
	}
	// 总和
	var total int
	var totalPrompt, totalCompletion, totalCalls int
	for _, r := range rows {
		total += r.Tokens
		totalPrompt += r.TokensPrompt
		totalCompletion += r.TokensCompletion
		totalCalls += r.CallCount
	}
	utils.OK(c, gin.H{
		"range":           c.Query("range"),
		"startAt":         toUnix(start),
		"endAt":           toUnix(end),
		"users":           rows,
		"totalTokens":     total,
		"totalPrompt":     totalPrompt,
		"totalCompletion": totalCompletion,
		"totalCalls":      totalCalls,
	})
}

// parseTimeRange 把 range=day|week|month（可选 +startDate/endDate）转成 [start, end)。
func parseTimeRange(rng, startDate, endDate string) (time.Time, time.Time, bool) {
	if startDate != "" || endDate != "" || rng != "" {
		// 显式日期优先
		if startDate != "" || endDate != "" {
			s, _ := time.ParseInLocation("2006-01-02", defaultStr(startDate, time.Now().Format("2006-01-02")), time.Local)
			e, _ := time.ParseInLocation("2006-01-02", defaultStr(endDate, time.Now().Format("2006-01-02")), time.Local)
			e = e.Add(24 * time.Hour)
			return s, e, true
		}
	}
	switch rng {
	case "day":
		now := time.Now().Local()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		end := start.Add(24 * time.Hour)
		return start, end, true
	case "week":
		now := time.Now().Local()
		// Go Sunday=0；让周一为第一天
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.Local)
		return monday, monday.Add(7 * 24 * time.Hour), true
	case "month":
		now := time.Now().Local()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		return start, end, true
	}
	return time.Time{}, time.Time{}, false
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
