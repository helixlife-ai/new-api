package controller

import (
	"bytes"
	"encoding/json"
	"export-service/config"
	"export-service/model"
	"export-service/service"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportController 导出控制器
type ExportController struct {
	cfg          *config.Config
	newAPIClient *service.NewAPIClient
}

// NewExportController 创建导出控制器
func NewExportController(cfg *config.Config) *ExportController {
	return &ExportController{
		cfg:          cfg,
		newAPIClient: service.NewNewAPIClient(cfg.NewAPIToolsURL),
	}
}

// PreviewRequest 预览请求
type PreviewRequest struct {
	FilterType     string   `json:"filter_type" binding:"required,oneof=feishu_csv token_name"`
	FeishuAppIDs   []string `json:"feishu_app_ids"`
	TokenName      string   `json:"token_name"`
	TimeRangeStart int64    `json:"time_range_start" binding:"required"`
	TimeRangeEnd   int64    `json:"time_range_end" binding:"required"`
}

// PreviewResponse 预览响应
type PreviewResponse struct {
	TokenCount  int `json:"token_count"`
	RowEstimate int `json:"row_estimate"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	FilterType     string   `json:"filter_type" binding:"required,oneof=feishu_csv token_name"`
	FeishuAppIDs   []string `json:"feishu_app_ids"`
	TokenName      string   `json:"token_name"`
	TimeRangeStart int64    `json:"time_range_start" binding:"required"`
	TimeRangeEnd   int64    `json:"time_range_end" binding:"required"`
}

// Preview 预览导出数据
func (c *ExportController) Preview(ctx *gin.Context) {
	var req PreviewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证时间范围
	if req.TimeRangeStart >= req.TimeRangeEnd {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid time range"})
		return
	}

	// 根据过滤类型获取token名称列表
	var tokenNames []string
	var tokenMappings []*model.FeishuTokenMapping

	switch req.FilterType {
	case "feishu_csv":
		if len(req.FeishuAppIDs) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "feishu_app_ids is required for feishu_csv filter"})
			return
		}
		for _, appID := range req.FeishuAppIDs {
			mapping, err := model.GetMappingByFeishuAppID(appID)
			if err != nil {
				continue // 跳过不存在的映射
			}
			tokenNames = append(tokenNames, mapping.TokenName)
			tokenMappings = append(tokenMappings, mapping)
		}
	case "token_name":
		if req.TokenName == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "token_name is required for token_name filter"})
			return
		}
		tokenNames = append(tokenNames, req.TokenName)
	}

	if len(tokenNames) == 0 {
		ctx.JSON(http.StatusOK, PreviewResponse{
			TokenCount:  0,
			RowEstimate: 0,
		})
		return
	}

	// 调用new-api获取预估行数
	rowEstimate, err := c.newAPIClient.EstimateLogCount(ctx, tokenNames, req.TimeRangeStart, req.TimeRangeEnd)
	if err != nil {
		// 如果API不支持估算，使用token数量作为粗略估计
		rowEstimate = len(tokenNames) * 100 // 粗略估计
	}

	ctx.JSON(http.StatusOK, PreviewResponse{
		TokenCount:  len(tokenNames),
		RowEstimate: rowEstimate,
	})
}

// Download 下载导出数据
func (c *ExportController) Download(ctx *gin.Context) {
	var req ExportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证时间范围
	if req.TimeRangeStart >= req.TimeRangeEnd {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid time range"})
		return
	}

	// 根据过滤类型获取token名称列表和映射
	var tokenNames []string
	tokenMappingMap := make(map[string]*model.FeishuTokenMapping)

	switch req.FilterType {
	case "feishu_csv":
		if len(req.FeishuAppIDs) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "feishu_app_ids is required for feishu_csv filter"})
			return
		}
		for _, appID := range req.FeishuAppIDs {
			mapping, err := model.GetMappingByFeishuAppID(appID)
			if err != nil {
				continue // 跳过不存在的映射
			}
			tokenNames = append(tokenNames, mapping.TokenName)
			tokenMappingMap[mapping.TokenName] = mapping
		}
	case "token_name":
		if req.TokenName == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "token_name is required for token_name filter"})
			return
		}
		tokenNames = append(tokenNames, req.TokenName)
	}

	if len(tokenNames) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no valid tokens found"})
		return
	}

	// 获取操作人信息
	operator := ctx.GetString("operator")
	if operator == "" {
		operator = "anonymous"
	}

	// 获取客户端IP
	sourceIP := ctx.ClientIP()

	// 构建文件名
	filename := fmt.Sprintf("export_%s_%s.csv",
		time.Now().Format("20060102_150405"),
		req.FilterType)

	// 创建临时缓冲区来存储CSV数据
	var buf bytes.Buffer

	// 写入UTF-8 BOM到缓冲区
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	// 创建CSV写入器（写入缓冲区）
	csvWriter := service.NewCSVWriter(&buf)

	// 写入表头
	if err := csvWriter.WriteHeader(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write CSV header"})
		return
	}

	// 流式拉取并写入数据
	rowCount := 0
	var streamErr error

	// 根据过滤类型选择流式获取方式
	if req.FilterType == "token_name" && len(tokenNames) == 1 {
		// 单token名称模式
		streamErr = c.newAPIClient.StreamLogsByTokenName(ctx, tokenNames[0], req.TimeRangeStart, req.TimeRangeEnd, func(log *service.LogEntry) error {
			record := c.buildCSVRecord(log, tokenMappingMap)
			if err := csvWriter.WriteRecord(record); err != nil {
				return err
			}
			rowCount++
			return nil
		})
	} else {
		// 多token名称模式（feishu_csv 或理论上支持的多token_name）
		streamErr = c.newAPIClient.StreamLogsByTokenNames(ctx, tokenNames, req.TimeRangeStart, req.TimeRangeEnd, func(log *service.LogEntry) error {
			record := c.buildCSVRecord(log, tokenMappingMap)
			if err := csvWriter.WriteRecord(record); err != nil {
				return err
			}
			rowCount++
			return nil
		})
	}

	// 刷新 csv.Writer 内部的 bufio 缓冲区
	csvWriter.Flush()
	if flushErr := csvWriter.Error(); flushErr != nil && streamErr == nil {
		streamErr = flushErr
	}

	status := "success"
	if streamErr != nil {
		status = "failed"
	}

	// 保存文件到磁盘（即使导出失败也保存，方便排查问题）
	filePath := ""
	if c.cfg.ExportStorageDir != "" {
		filePath = filepath.Join(c.cfg.ExportStorageDir, filename)
		if writeErr := os.WriteFile(filePath, buf.Bytes(), 0644); writeErr != nil {
			// 记录错误但不中断流程
			fmt.Printf("Failed to save export file: %v\n", writeErr)
			filePath = "" // 保存失败则清空路径
		}
	}

	// 记录导出日志
	filterDetail, _ := json.Marshal(map[string]interface{}{
		"feishu_app_ids": req.FeishuAppIDs,
		"token_name":     req.TokenName,
	})

	exportLog := &model.ExportLog{
		Operator:       operator,
		FilterType:     req.FilterType,
		FilterDetail:   string(filterDetail),
		TimeRangeStart: req.TimeRangeStart,
		TimeRangeEnd:   req.TimeRangeEnd,
		TokenCount:     len(tokenNames),
		RowCount:       rowCount,
		Status:         status,
		SourceIP:       sourceIP,
		FilePath:       filePath,
	}
	_ = model.CreateExportLog(exportLog)

	// 如果流式导出出错，返回错误
	if streamErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": streamErr.Error()})
		return
	}

	// 设置响应头并返回文件内容
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// buildCSVRecord 构建CSV记录
func (c *ExportController) buildCSVRecord(log *service.LogEntry, tokenMappingMap map[string]*model.FeishuTokenMapping) *service.CSVRecord {
	// 查找对应的飞书映射（通过token_name）
	mapping := tokenMappingMap[log.TokenName]
	if mapping == nil {
		// 如果没有映射，使用默认值
		mapping = &model.FeishuTokenMapping{
			FeishuAppID: "",
			FeishuName:  "",
			TokenName:   log.TokenName,
		}
	}

	return &service.CSVRecord{
		CreatedAt:           time.Unix(log.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		TokenName:           mapping.TokenName,
		FeishuAppID:         mapping.FeishuAppID,
		FeishuName:          mapping.FeishuName,
		ModelName:           log.ModelName,
		PromptTokens:        log.PromptTokens,
		CompletionTokens:    log.CompletionTokens,
		CacheTokens:         log.CacheTokens,
		CacheCreationTokens: log.CacheCreationTokens,
	}
}

// ExportLogResponse 导出日志响应（前端友好格式）
type ExportLogResponse struct {
	ID           uint   `json:"id"`
	Operator     string `json:"operator"`
	FilterType   string `json:"filterType"`
	FilterDetail string `json:"filterDetail"`
	StartTime    int64  `json:"startTime"`
	EndTime      int64  `json:"endTime"`
	TokenCount   int    `json:"tokenCount"`
	RecordCount  int    `json:"recordCount"`
	Status       string `json:"status"`
	SourceIP     string `json:"sourceIp"`
	CreatedAt    string `json:"createdAt"`
	DownloadURL  string `json:"downloadUrl"`
}

// toExportLogResponse 将模型转换为前端响应格式
func toExportLogResponse(log *model.ExportLog) *ExportLogResponse {
	downloadURL := ""
	if log.FilePath != "" && log.Status == "success" {
		downloadURL = fmt.Sprintf("/api/export/download-file/%d", log.ID)
	}
	return &ExportLogResponse{
		ID:           log.ID,
		Operator:     log.Operator,
		FilterType:   log.FilterType,
		FilterDetail: log.FilterDetail,
		StartTime:    log.TimeRangeStart,
		EndTime:      log.TimeRangeEnd,
		TokenCount:   log.TokenCount,
		RecordCount:  log.RowCount,
		Status:       log.Status,
		SourceIP:     log.SourceIP,
		CreatedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
		DownloadURL:  downloadURL,
	}
}

// GetExportLogs 获取导出日志列表
func (c *ExportController) GetExportLogs(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 获取时间范围参数（前端使用驼峰命名）
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")

	// 兼容下划线命名（如有需要）
	if startTimeStr == "" {
		startTimeStr = ctx.Query("start_time")
	}
	if endTimeStr == "" {
		endTimeStr = ctx.Query("end_time")
	}

	var logs []model.ExportLog
	var err error

	if startTimeStr != "" && endTimeStr != "" {
		startTime, _ := strconv.ParseInt(startTimeStr, 10, 64)
		endTime, _ := strconv.ParseInt(endTimeStr, 10, 64)
		logs, err = model.GetExportLogsWithTimeRange(startTime, endTime, pageSize, offset)
	} else {
		logs, err = model.GetExportLogs(pageSize, offset)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为前端友好的格式
	var response []*ExportLogResponse
	for i := range logs {
		response = append(response, toExportLogResponse(&logs[i]))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":     response,
		"page":     page,
		"pageSize": pageSize,
	})
}

// DownloadFile 下载历史导出文件
func (c *ExportController) DownloadFile(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid export log id"})
		return
	}

	// 获取导出日志
	log, err := model.GetExportLogByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "export log not found"})
		return
	}

	// 检查是否有文件
	if log.FilePath == "" {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found for this export"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(log.FilePath); os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file has been deleted or expired"})
		return
	}

	// 打开文件
	file, err := os.Open(log.FilePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer file.Close()

	// 提取文件名
	filename := filepath.Base(log.FilePath)

	// 设置响应头
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// 流式传输文件内容
	_, err = io.Copy(ctx.Writer, file)
	if err != nil {
		// 流式传输已经开始，无法返回JSON错误
		fmt.Printf("Failed to stream file: %v\n", err)
	}
}
