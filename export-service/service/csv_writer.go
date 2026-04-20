package service

import (
	"encoding/csv"
	"io"
)

// CSVRecord CSV记录
type CSVRecord struct {
	CreatedAt           string
	TokenName           string
	FeishuAppID         string
	FeishuName          string
	ModelName           string
	PromptTokens        int
	CompletionTokens    int
	CacheTokens         int
	CacheCreationTokens int
}

// CSVWriter CSV写入器
type CSVWriter struct {
	writer *csv.Writer
}

// CSV表头
var csvHeaders = []string{
	"created_at",
	"token_name",
	"feishu_app_id",
	"feishu_name",
	"model_name",
	"prompt_tokens",
	"completion_tokens",
	"cache_tokens",
	"cache_creation_tokens",
}

// NewCSVWriter 创建CSV写入器
func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{
		writer: csv.NewWriter(w),
	}
}

// WriteHeader 写入表头
func (w *CSVWriter) WriteHeader() error {
	return w.writer.Write(csvHeaders)
}

// WriteRecord 写入记录
func (w *CSVWriter) WriteRecord(record *CSVRecord) error {
	row := []string{
		record.CreatedAt,
		record.TokenName,
		record.FeishuAppID,
		record.FeishuName,
		record.ModelName,
		intToString(record.PromptTokens),
		intToString(record.CompletionTokens),
		intToString(record.CacheTokens),
		intToString(record.CacheCreationTokens),
	}
	return w.writer.Write(row)
}

// WriteRecords 批量写入记录
func (w *CSVWriter) WriteRecords(records []*CSVRecord) error {
	for _, record := range records {
		if err := w.WriteRecord(record); err != nil {
			return err
		}
	}
	return nil
}

// Flush 刷新缓冲区
func (w *CSVWriter) Flush() {
	w.writer.Flush()
}

// Error 返回写入错误
func (w *CSVWriter) Error() error {
	return w.writer.Error()
}

// intToString 将int转换为string
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	// 预分配足够的空间
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
