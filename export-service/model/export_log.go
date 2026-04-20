package model

import (
	"time"
)

// ExportLog 导出操作日志
type ExportLog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	Operator       string    `gorm:"size:64" json:"operator"`
	FilterType     string    `gorm:"size:32" json:"filter_type"`     // 'feishu_csv' 或 'token_name'
	FilterDetail   string    `gorm:"type:text" json:"filter_detail"` // JSON
	TimeRangeStart int64     `json:"time_range_start"`
	TimeRangeEnd   int64     `json:"time_range_end"`
	TokenCount     int       `json:"token_count"`
	RowCount       int       `json:"row_count"`
	Status         string    `gorm:"size:16;default:'success'" json:"status"` // 'success', 'failed', 'processing'
	SourceIP       string    `gorm:"size:64" json:"source_ip"`
	FilePath       string    `gorm:"size:512" json:"file_path"` // 导出文件路径
	CreatedAt      time.Time `json:"created_at"`
}

// TableName 指定表名
func (ExportLog) TableName() string {
	return "export_logs"
}

// CreateExportLog 创建导出日志
func CreateExportLog(log *ExportLog) error {
	log.CreatedAt = time.Now()
	return DB.Create(log).Error
}

// GetExportLogs 获取导出日志列表
func GetExportLogs(limit, offset int) ([]ExportLog, error) {
	var logs []ExportLog
	if err := DB.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetExportLogsWithTimeRange 根据时间范围获取导出日志列表
func GetExportLogsWithTimeRange(startTime, endTime int64, limit, offset int) ([]ExportLog, error) {
	var logs []ExportLog
	query := DB.Order("created_at DESC")

	if startTime > 0 {
		query = query.Where("time_range_start >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("time_range_end <= ?", endTime)
	}

	if err := query.Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetExportLogByID 根据ID获取导出日志
func GetExportLogByID(id uint) (*ExportLog, error) {
	var log ExportLog
	if err := DB.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
