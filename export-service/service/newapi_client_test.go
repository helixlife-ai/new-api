package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// buildMockServer 创建一个 mock newapi-tools /api/logs 服务（页码分页）
func buildMockServer(t *testing.T, allLogs []LogEntry, pageSize int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/logs" {
			http.NotFound(w, r)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		ps, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if ps < 1 {
			ps = pageSize
		}

		start := (page - 1) * ps
		end := start + ps
		if end > len(allLogs) {
			end = len(allLogs)
		}
		var items []LogEntry
		if start < len(allLogs) {
			items = allLogs[start:end]
		} else {
			items = []LogEntry{}
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"items":     items,
				"total":     len(allLogs),
				"page":      page,
				"page_size": ps,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	return srv
}

// ─── T1: 单页结果 ─────────────────────────────────────────────────────────────

func TestNewAPIClient_StreamLogsByTokenName_SinglePage(t *testing.T) {
	logs := []LogEntry{
		{ID: 1, TokenName: "tok", ModelName: "gpt-4", PromptTokens: 10, Quota: 100, CreatedAt: 1713000000},
		{ID: 2, TokenName: "tok", ModelName: "gpt-4", PromptTokens: 20, Quota: 200, CreatedAt: 1713000001},
	}
	srv := buildMockServer(t, logs, 100)
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)

	var collected []LogEntry
	err := client.StreamLogsByTokenName(context.Background(), "tok", 1712000000, 1714000000, func(log *LogEntry) error {
		collected = append(collected, *log)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamLogsByTokenName error: %v", err)
	}
	if len(collected) != 2 {
		t.Errorf("expected 2 logs, got %d", len(collected))
	}
}

// ─── T2: 多页翻页 ─────────────────────────────────────────────────────────────

func TestNewAPIClient_StreamLogsByTokenName_MultiPage(t *testing.T) {
	// 构造 250 条日志，streamPageSize=100 需要翻 3 页
	logs := make([]LogEntry, 250)
	for i := range logs {
		logs[i] = LogEntry{ID: i + 1, TokenName: "tok", Quota: i * 10, CreatedAt: int64(1713000000 + i)}
	}
	srv := buildMockServer(t, logs, streamPageSize)
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)

	var collected []LogEntry
	err := client.StreamLogsByTokenName(context.Background(), "tok", 1712000000, 1714000000, func(log *LogEntry) error {
		collected = append(collected, *log)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamLogsByTokenName error: %v", err)
	}
	if len(collected) != 250 {
		t.Errorf("expected 250 logs across pages, got %d", len(collected))
	}

	// 验证无重复
	seen := make(map[int]bool)
	for _, l := range collected {
		if seen[l.ID] {
			t.Errorf("duplicate ID %d", l.ID)
		}
		seen[l.ID] = true
	}
}

// ─── T3: 多 token 名称逐个查询 ────────────────────────────────────────────────

func TestNewAPIClient_StreamLogsByTokenNames_MultipleTokens(t *testing.T) {
	logs := []LogEntry{
		{ID: 1, TokenName: "alpha", Quota: 10, CreatedAt: 1713000000},
		{ID: 2, TokenName: "beta", Quota: 20, CreatedAt: 1713000001},
		{ID: 3, TokenName: "gamma", Quota: 30, CreatedAt: 1713000002},
	}
	srv := buildMockServer(t, logs, 100)
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)

	var collected []LogEntry
	// 每个 token 各调用一次，mock 返回全部数据（不按 token 过滤）
	err := client.StreamLogsByTokenNames(context.Background(), []string{"alpha", "beta", "gamma"}, 1712000000, 1714000000, func(log *LogEntry) error {
		collected = append(collected, *log)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamLogsByTokenNames error: %v", err)
	}
	// 3 个 token × 3 条记录（mock 不过滤）= 9 条
	if len(collected) != 9 {
		t.Errorf("expected 9 logs (3 tokens × 3 records each from mock), got %d", len(collected))
	}
}

// ─── T4: 回调返回错误时立即终止 ─────────────────────────────────────────────

func TestNewAPIClient_StreamLogsByTokenName_CallbackError(t *testing.T) {
	logs := make([]LogEntry, 5)
	for i := range logs {
		logs[i] = LogEntry{ID: i + 1, TokenName: "tok", CreatedAt: int64(1713000000 + i)}
	}
	srv := buildMockServer(t, logs, 100)
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)

	count := 0
	err := client.StreamLogsByTokenName(context.Background(), "tok", 1712000000, 1714000000, func(log *LogEntry) error {
		count++
		if count >= 2 {
			return fmt.Errorf("stop after 2")
		}
		return nil
	})
	if err == nil {
		t.Error("expected error from callback, got nil")
	}
	if count != 2 {
		t.Errorf("expected callback called 2 times before stop, got %d", count)
	}
}

// ─── T5: 服务器错误处理 ──────────────────────────────────────────────────────

func TestNewAPIClient_StreamLogsByTokenName_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)
	err := client.StreamLogsByTokenName(context.Background(), "tok", 1712000000, 1714000000, func(log *LogEntry) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

// ─── T6: EstimateLogCount 从 total 字段读取 ──────────────────────────────────

func TestNewAPIClient_EstimateLogCount(t *testing.T) {
	logs := make([]LogEntry, 42)
	for i := range logs {
		logs[i] = LogEntry{ID: i + 1, TokenName: "tok", CreatedAt: int64(1713000000 + i)}
	}
	srv := buildMockServer(t, logs, 100)
	defer srv.Close()

	client := NewNewAPIClient(srv.URL)
	count, err := client.EstimateLogCount(context.Background(), []string{"tok"}, 1712000000, 1714000000)
	if err != nil {
		t.Fatalf("EstimateLogCount error: %v", err)
	}
	if count != 42 {
		t.Errorf("expected 42, got %d", count)
	}
}
