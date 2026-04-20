package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"newapi-tools/model"
	"newapi-tools/service"

	"github.com/gin-gonic/gin"
)

type LogController struct{}

func NewLogController() *LogController {
	return &LogController{}
}

// QueryLogs handles GET /api/logs — JSON paginated query.
func (c *LogController) QueryLogs(ctx *gin.Context) {
	startTimestamp, endTimestamp, err := parseTimeRange(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenName := ctx.Query("token_name")
	modelName := ctx.Query("model_name")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := model.GetLogsPaginated(startTimestamp, endTimestamp, tokenName, modelName, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with parsed Other fields
	type LogItem struct {
		model.Log
		CacheTokens         int `json:"cache_tokens"`
		CacheCreationTokens int `json:"cache_creation_tokens"`
		TotalTokens         int `json:"total_tokens"`
	}

	items := make([]LogItem, len(logs))
	for i := range logs {
		other := logs[i].ParseOther()
		items[i] = LogItem{
			Log:                 logs[i],
			CacheTokens:         other.CacheTokens,
			CacheCreationTokens: other.CacheCreationTokens,
			TotalTokens:         logs[i].PromptTokens + logs[i].CompletionTokens,
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ExportLogs handles GET /api/logs/export — CSV file download.
func (c *LogController) ExportLogs(ctx *gin.Context) {
	startTimestamp, endTimestamp, err := parseTimeRange(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenName := ctx.Query("token_name")
	modelName := ctx.Query("model_name")

	filename := fmt.Sprintf("logs_export_%s.csv", time.Now().Format("20060102_150405"))

	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// Write UTF-8 BOM for Excel compatibility
	ctx.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	csvWriter := service.NewCSVWriter(ctx.Writer)
	if err := csvWriter.WriteHeader(); err != nil {
		return
	}

	err = model.StreamLogs(ctx.Request.Context(), startTimestamp, endTimestamp, tokenName, modelName, func(log *model.Log) error {
		other := log.ParseOther()
		record := &service.CSVRecord{
			CreatedAt:           time.Unix(log.CreatedAt, 0).Format("2006-01-02 15:04:05"),
			TokenName:           log.TokenName,
			ModelName:           log.ModelName,
			PromptTokens:        log.PromptTokens,
			CompletionTokens:    log.CompletionTokens,
			CacheTokens:         other.CacheTokens,
			CacheCreationTokens: other.CacheCreationTokens,
			TotalTokens:         log.PromptTokens + log.CompletionTokens,
			Quota:               log.Quota,
			UseTime:             log.UseTime,
		}
		return csvWriter.WriteRecord(record)
	})

	csvWriter.Flush()

	if err != nil {
		fmt.Printf("Export stream error: %v\n", err)
	}
}

func parseTimeRange(ctx *gin.Context) (int64, int64, error) {
	startStr := ctx.Query("start_timestamp")
	endStr := ctx.Query("end_timestamp")
	if startStr == "" || endStr == "" {
		return 0, 0, fmt.Errorf("start_timestamp and end_timestamp are required")
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start_timestamp")
	}
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end_timestamp")
	}
	if start >= end {
		return 0, 0, fmt.Errorf("start_timestamp must be less than end_timestamp")
	}
	return start, end, nil
}
