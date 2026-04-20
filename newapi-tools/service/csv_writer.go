package service

import (
	"encoding/csv"
	"io"
)

var csvHeaders = []string{
	"created_at",
	"token_name",
	"model_name",
	"prompt_tokens",
	"completion_tokens",
	"cache_tokens",
	"cache_creation_tokens",
	"total_tokens",
	"quota",
	"use_time",
}

type CSVRecord struct {
	CreatedAt           string
	TokenName           string
	ModelName           string
	PromptTokens        int
	CompletionTokens    int
	CacheTokens         int
	CacheCreationTokens int
	TotalTokens         int
	Quota               int
	UseTime             int
}

type CSVWriter struct {
	writer *csv.Writer
}

func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{writer: csv.NewWriter(w)}
}

func (w *CSVWriter) WriteHeader() error {
	return w.writer.Write(csvHeaders)
}

func (w *CSVWriter) WriteRecord(r *CSVRecord) error {
	row := []string{
		r.CreatedAt,
		r.TokenName,
		r.ModelName,
		intToString(r.PromptTokens),
		intToString(r.CompletionTokens),
		intToString(r.CacheTokens),
		intToString(r.CacheCreationTokens),
		intToString(r.TotalTokens),
		intToString(r.Quota),
		intToString(r.UseTime),
	}
	return w.writer.Write(row)
}

func (w *CSVWriter) Flush() {
	w.writer.Flush()
}

func (w *CSVWriter) Error() error {
	return w.writer.Error()
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
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
