package controller

import (
	"export-service/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// MappingController 映射控制器
type MappingController struct{}

// NewMappingController 创建映射控制器
func NewMappingController() *MappingController {
	return &MappingController{}
}

// MappingResponse 映射响应（前端友好格式）
type MappingResponse struct {
	ID          uint     `json:"id"`
	FeishuAppID string   `json:"feishuAppId"`
	FeishuName  string   `json:"feishuName"`
	TokenID     int      `json:"tokenId"`
	TokenName   string   `json:"tokenName"`
	Tags        []string `json:"tags"`
	CreatedAt   string   `json:"createdAt"`
}

// toMappingResponse 将模型转换为前端响应格式
func toMappingResponse(m *model.FeishuTokenMapping) *MappingResponse {
	tags := []string{}
	if m.Labels != "" {
		// 按逗号分割标签
		for _, tag := range strings.Split(m.Labels, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return &MappingResponse{
		ID:          m.ID,
		FeishuAppID: m.FeishuAppID,
		FeishuName:  m.FeishuName,
		TokenID:     m.TokenID,
		TokenName:   m.TokenName,
		Tags:        tags,
		CreatedAt:   m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListMappingsRequest 查询映射请求
type ListMappingsRequest struct {
	Label string `form:"label"`
}

// CreateMappingRequest 创建映射请求（前端友好格式）
type CreateMappingRequest struct {
	FeishuAppID string   `json:"feishuAppId" binding:"required"`
	FeishuName  string   `json:"feishuName" binding:"required"`
	TokenID     int      `json:"tokenId" binding:"required"`
	TokenName   string   `json:"tokenName" binding:"required"`
	Tags        []string `json:"tags"`
}

// UpdateMappingRequest 更新映射请求（前端友好格式）
type UpdateMappingRequest struct {
	FeishuAppID string   `json:"feishuAppId"`
	FeishuName  string   `json:"feishuName"`
	TokenID     int      `json:"tokenId"`
	TokenName   string   `json:"tokenName"`
	Tags        []string `json:"tags"`
}

// ImportMappingsResponse 导入响应
type ImportMappingsResponse struct {
	ImportedCount int    `json:"imported_count"`
	Message       string `json:"message"`
}

// List 获取映射列表
func (c *MappingController) List(ctx *gin.Context) {
	var req ListMappingsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mappings []model.FeishuTokenMapping
	var err error

	if req.Label != "" {
		mappings, err = model.GetMappingsByLabel(req.Label)
	} else {
		mappings, err = model.GetAllMappings()
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为前端友好的格式
	var response []*MappingResponse
	for i := range mappings {
		response = append(response, toMappingResponse(&mappings[i]))
	}

	ctx.JSON(http.StatusOK, response)
}

// Create 创建映射
func (c *MappingController) Create(ctx *gin.Context) {
	var req CreateMappingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 将 tags 转换为 labels 字符串
	labels := strings.Join(req.Tags, ",")

	mapping := &model.FeishuTokenMapping{
		FeishuAppID: req.FeishuAppID,
		FeishuName:  req.FeishuName,
		TokenID:     req.TokenID,
		TokenName:   req.TokenName,
		Labels:      labels,
	}

	if err := model.CreateMapping(mapping); err != nil {
		// 检查是否是唯一性冲突（大小写不敏感，兼容 PostgreSQL/SQLite/MySQL 错误格式）
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "unique") || strings.Contains(errMsg, "duplicate") {
			ctx.JSON(http.StatusConflict, gin.H{"error": "mapping already exists for this feishu_app_id or token_id"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, toMappingResponse(mapping))
}

// Update 更新映射
func (c *MappingController) Update(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateMappingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查映射是否存在
	_, err = model.GetMappingByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "mapping not found"})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.FeishuAppID != "" {
		updates["feishu_app_id"] = req.FeishuAppID
	}
	if req.FeishuName != "" {
		updates["feishu_name"] = req.FeishuName
	}
	if req.TokenID != 0 {
		updates["token_id"] = req.TokenID
	}
	if req.TokenName != "" {
		updates["token_name"] = req.TokenName
	}
	// Tags 转换为 Labels 字符串
	updates["labels"] = strings.Join(req.Tags, ",")

	if err := model.UpdateMapping(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的数据
	updated, _ := model.GetMappingByID(uint(id))
	if updated != nil {
		ctx.JSON(http.StatusOK, toMappingResponse(updated))
	} else {
		ctx.JSON(http.StatusOK, gin.H{"message": "update successful"})
	}
}

// Delete 删除映射
func (c *MappingController) Delete(ctx *gin.Context) {
	// 检查是否有label查询参数（批量删除）
	label := ctx.Query("label")
	if label != "" {
		if err := model.DeleteMappingsByLabel(label); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"message": "mappings deleted successfully", "label": label})
		return
	}

	// 单条删除
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 检查映射是否存在
	if _, err := model.GetMappingByID(uint(id)); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "mapping not found"})
		return
	}

	if err := model.DeleteMapping(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "mapping deleted successfully"})
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	Tags []string `json:"tags" binding:"required"`
}

// BatchDelete 批量删除映射（根据标签精确匹配）
func (c *MappingController) BatchDelete(ctx *gin.Context) {
	var req BatchDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Tags) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "tags are required"})
		return
	}

	// 使用精确匹配删除，避免LIKE模糊匹配导致的误删
	totalDeleted, err := model.DeleteMappingsByLabelsExact(req.Tags)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "mappings deleted successfully",
		"tags":    req.Tags,
		"count":   totalDeleted,
	})
}

// BatchImportRequest 批量导入请求
type BatchImportRequest struct {
	Mappings []struct {
		FeishuAppID string   `json:"feishuAppId" binding:"required"`
		FeishuName  string   `json:"feishuName"`
		TokenID     int      `json:"tokenId"`
		TokenName   string   `json:"tokenName" binding:"required"`
		Tags        []string `json:"tags"`
	} `json:"mappings" binding:"required"`
}

// BatchImport 批量导入映射（从前端JSON）
func (c *MappingController) BatchImport(ctx *gin.Context) {
	var req BatchImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var importedCount int
	for _, m := range req.Mappings {
		// 尝试根据token_name获取token_id（这里简化处理，实际可能需要调用new-api）
		tokenID := m.TokenID
		if tokenID == 0 && m.TokenName != "" {
			// 如果token_id为0但有token_name，尝试查找
			// 这里简化处理，实际可能需要调用new-api的接口
		}

		labels := strings.Join(m.Tags, ",")
		mapping := &model.FeishuTokenMapping{
			FeishuAppID: m.FeishuAppID,
			FeishuName:  m.FeishuName,
			TokenID:     tokenID,
			TokenName:   m.TokenName,
			Labels:      labels,
		}

		// 检查是否已存在
		existing, _ := model.GetMappingByFeishuAppID(m.FeishuAppID)
		if existing != nil {
			// 更新现有记录
			updates := map[string]interface{}{
				"feishu_name": m.FeishuName,
				"token_id":    tokenID,
				"token_name":  m.TokenName,
				"labels":      labels,
			}
			if err := model.UpdateMapping(existing.ID, updates); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update mapping: " + err.Error()})
				return
			}
		} else {
			// 创建新记录
			if err := model.CreateMapping(mapping); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mapping: " + err.Error()})
				return
			}
		}
		importedCount++
	}

	ctx.JSON(http.StatusOK, ImportMappingsResponse{
		ImportedCount: importedCount,
		Message:       "import completed successfully",
	})
}

// Import 从CSV导入映射
func (c *MappingController) Import(ctx *gin.Context) {
	// 获取labels参数（可选，用于为导入的数据添加标签）
	labels := ctx.PostForm("labels")

	// 获取上传的文件
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	count, err := model.ImportMappingsFromCSV(file, labels)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, ImportMappingsResponse{
		ImportedCount: count,
		Message:       "import completed successfully",
	})
}
