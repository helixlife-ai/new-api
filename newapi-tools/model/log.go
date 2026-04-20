package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const LogTypeConsume = 2

type Log struct {
	Id               int    `json:"id" gorm:"column:id"`
	CreatedAt        int64  `json:"created_at" gorm:"column:created_at"`
	Type             int    `json:"type" gorm:"column:type"`
	TokenName        string `json:"token_name" gorm:"column:token_name"`
	ModelName        string `json:"model_name" gorm:"column:model_name"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"column:prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens" gorm:"column:completion_tokens"`
	Quota            int    `json:"quota" gorm:"column:quota"`
	UseTime          int    `json:"use_time" gorm:"column:use_time"`
	Other            string `json:"other" gorm:"column:other"`
}

func (Log) TableName() string { return "logs" }

type OtherInfo struct {
	CacheTokens         int `json:"cache_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

func (l *Log) ParseOther() OtherInfo {
	var info OtherInfo
	if l.Other != "" {
		_ = json.Unmarshal([]byte(l.Other), &info)
	}
	return info
}

// sanitizeLikePattern escapes special LIKE characters using '!' as the escape char.
func sanitizeLikePattern(input string) string {
	s := strings.ReplaceAll(input, "!", "!!")
	s = strings.ReplaceAll(s, "%", "!%")
	s = strings.ReplaceAll(s, "_", "!_")
	return "%" + s + "%"
}

func buildBaseQuery(db *gorm.DB, startTimestamp, endTimestamp int64, tokenName, modelName string) *gorm.DB {
	tx := db.Where("type = ?", LogTypeConsume)
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if tokenName != "" {
		tx = tx.Where("token_name LIKE ? ESCAPE '!'", sanitizeLikePattern(tokenName))
	}
	if modelName != "" {
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", sanitizeLikePattern(modelName))
	}
	return tx
}

// GetLogsPaginated returns logs with offset-based pagination for JSON API.
func GetLogsPaginated(startTimestamp, endTimestamp int64, tokenName, modelName string, page, pageSize int) (logs []Log, total int64, err error) {
	tx := buildBaseQuery(DB.Model(&Log{}), startTimestamp, endTimestamp, tokenName, modelName)

	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	offset := (page - 1) * pageSize
	err = tx.Order("id desc").Offset(offset).Limit(pageSize).Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	return logs, total, nil
}

// StreamLogs iterates over all matching logs using cursor-based pagination
// and calls the callback for each row. Used for CSV export.
func StreamLogs(ctx context.Context, startTimestamp, endTimestamp int64, tokenName, modelName string, callback func(*Log) error) error {
	const batchSize = 1000
	cursor := 0

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var batch []Log
		tx := buildBaseQuery(DB.WithContext(ctx), startTimestamp, endTimestamp, tokenName, modelName)
		if cursor > 0 {
			tx = tx.Where("id < ?", cursor)
		}
		if err := tx.Order("id desc").Limit(batchSize).Find(&batch).Error; err != nil {
			return fmt.Errorf("stream query failed: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		for i := range batch {
			if err := callback(&batch[i]); err != nil {
				return err
			}
		}

		cursor = batch[len(batch)-1].Id
		if len(batch) < batchSize {
			break
		}
	}
	return nil
}
