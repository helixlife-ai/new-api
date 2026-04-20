package controller_test

import (
	"encoding/json"
	"export-service/controller"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── T1: 健康检查 ─────────────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health check failed: %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp)
	}
}

// ─── T2: Preview 缺少必填参数返回 400 ──────────────────────────────────────

func TestExport_Preview_MissingParams(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	body := `{"time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing filter_type, got %d", w.Code)
	}
}

// ─── T3: Preview token_name 模式返回正确结构 ─────────────────────────────────

func TestExport_Preview_TokenName(t *testing.T) {
	setupTestDB(t)

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{"items": []interface{}{}, "total": 0, "page": 1, "page_size": 100},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockSrv.Close()

	r := newTestRouter(t, mockSrv.URL)

	body := `{"filter_type":"token_name","token_name":"test%","time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", w.Code, w.Body.String())
	}

	var resp controller.PreviewResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TokenCount != 1 {
		t.Errorf("expected token_count=1, got %d", resp.TokenCount)
	}
}

// ─── T4: Preview feishu_csv 空映射表返回 token_count=0 ──────────────────────

func TestExport_Preview_FeishuCsv_NoMapping(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	body := `{"filter_type":"feishu_csv","feishu_app_ids":["cli_notexist"],"time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("preview failed: %d %s", w.Code, w.Body.String())
	}
	var resp controller.PreviewResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TokenCount != 0 {
		t.Errorf("expected token_count=0 for unknown feishu_app_id, got %d", resp.TokenCount)
	}
}

// ─── T5: Download 返回 UTF-8 BOM 和正确 Content-Type ────────────────────────

func TestExport_Download_BOMAndContentType(t *testing.T) {
	setupTestDB(t)

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": 1, "token_name": "test-tok", "model_name": "gpt-4",
						"prompt_tokens": 10, "completion_tokens": 5,
						"cache_tokens": 0, "cache_creation_tokens": 0,
						"total_tokens": 15, "quota": 100, "use_time": 500,
						"created_at": 1713000000},
				},
				"total": 1, "page": 1, "page_size": 100,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockSrv.Close()

	r := newTestRouter(t, mockSrv.URL)

	body := `{"filter_type":"token_name","token_name":"test-tok","time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download failed: %d %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected Content-Type=text/csv, got %q", ct)
	}

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("expected attachment in Content-Disposition, got %q", cd)
	}

	bodyBytes := w.Body.Bytes()
	if len(bodyBytes) < 3 {
		t.Fatal("response body too short")
	}
	if bodyBytes[0] != 0xEF || bodyBytes[1] != 0xBB || bodyBytes[2] != 0xBF {
		t.Errorf("expected UTF-8 BOM, got %x %x %x", bodyBytes[0], bodyBytes[1], bodyBytes[2])
	}
}

// ─── T6: Download 验证 CSV 表头列正确 ──────────────────────────────────────

func TestExport_Download_CSVHeaders(t *testing.T) {
	setupTestDB(t)

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{"items": []interface{}{}, "total": 0, "page": 1, "page_size": 100},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockSrv.Close()

	r := newTestRouter(t, mockSrv.URL)

	body := `{"filter_type":"token_name","token_name":"empty%","time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download failed: %d %s", w.Code, w.Body.String())
	}

	csvContent := string(w.Body.Bytes()[3:]) // 跳过 BOM
	lines := strings.Split(strings.TrimSpace(csvContent), "\n")
	if len(lines) == 0 {
		t.Fatal("CSV response is empty (no header)")
	}

	expected := "created_at,token_name,feishu_app_id,feishu_name,model_name,prompt_tokens,completion_tokens,cache_tokens,cache_creation_tokens"
	headerLine := strings.TrimRight(lines[0], "\r")
	if headerLine != expected {
		t.Errorf("CSV header mismatch:\n  expected: %s\n  got:      %s", expected, headerLine)
	}
}

// ─── T7: Download Flush 缺失 Bug 验证 ──────────────────────────────────────
//
// csv.Writer 使用 bufio.Writer，写入后需显式调用 Flush()。
// controller/export.go Download() 函数未调用 csvWriter.Flush()，
// 可能导致最后一批数据滞留在缓冲区中。

func TestExport_Download_FlushBug(t *testing.T) {
	setupTestDB(t)

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []map[string]interface{}{
			{"id": 1, "token_name": "tok", "model_name": "gpt-4", "prompt_tokens": 10,
				"completion_tokens": 5, "cache_tokens": 0, "cache_creation_tokens": 0,
				"total_tokens": 15, "quota": 100, "use_time": 100, "created_at": 1713000000},
			{"id": 2, "token_name": "tok", "model_name": "gpt-4", "prompt_tokens": 20,
				"completion_tokens": 8, "cache_tokens": 0, "cache_creation_tokens": 0,
				"total_tokens": 28, "quota": 200, "use_time": 200, "created_at": 1713000001},
			{"id": 3, "token_name": "tok", "model_name": "gpt-4", "prompt_tokens": 30,
				"completion_tokens": 12, "cache_tokens": 5, "cache_creation_tokens": 2,
				"total_tokens": 47, "quota": 300, "use_time": 300, "created_at": 1713000002},
		}
		resp := map[string]interface{}{
			"data": map[string]interface{}{"items": items, "total": 3, "page": 1, "page_size": 100},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockSrv.Close()

	r := newTestRouter(t, mockSrv.URL)

	body := `{"filter_type":"token_name","token_name":"tok","time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download failed: %d %s", w.Code, w.Body.String())
	}

	bodyBytes := w.Body.Bytes()
	csvContent := string(bodyBytes[3:]) // 跳过 BOM

	lines := strings.Split(strings.TrimSpace(csvContent), "\n")
	nonEmptyLines := 0
	for _, l := range lines {
		if strings.TrimRight(l, "\r") != "" {
			nonEmptyLines++
		}
	}

	t.Logf("CSV line count: %d (expected 4: 1 header + 3 records)", nonEmptyLines)
	t.Logf("CSV:\n%s", csvContent)

	if nonEmptyLines < 4 {
		t.Errorf("BUG CONFIRMED: expected 4 lines (1 header + 3 records), got %d — controller/export.go Download() 缺少 csvWriter.Flush()", nonEmptyLines)
	} else {
		t.Log("PASS: all records flushed to response")
	}
}

// ─── T8: Download feishu_csv 模式映射关联 ────────────────────────────────────

func TestExport_Download_FeishuCsvMode(t *testing.T) {
	setupTestDB(t)

	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []map[string]interface{}{
			{"id": 1, "token_name": "feishu-tok", "model_name": "gpt-4",
				"prompt_tokens": 10, "completion_tokens": 5, "cache_tokens": 0,
				"cache_creation_tokens": 0, "total_tokens": 15, "quota": 100,
				"use_time": 500, "created_at": 1713000000},
		}
		resp := map[string]interface{}{
			"data": map[string]interface{}{"items": items, "total": 1, "page": 1, "page_size": 100},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockSrv.Close()

	r := newTestRouter(t, mockSrv.URL)

	// 先创建飞书映射
	createBody := `{"feishuAppId":"cli_ftest","feishuName":"飞书应用","tokenId":999,"tokenName":"feishu-tok"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("mapping create failed: %s", createW.Body.String())
	}

	// 导出
	body := `{"filter_type":"feishu_csv","feishu_app_ids":["cli_ftest"],"time_range_start":1712764800,"time_range_end":1713369600}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/download", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download failed: %d %s", w.Code, w.Body.String())
	}

	csvContent := string(w.Body.Bytes()[3:])
	t.Logf("CSV:\n%s", csvContent)

	if !strings.Contains(csvContent, "cli_ftest") {
		t.Error("expected feishu_app_id 'cli_ftest' in CSV")
	}
	if !strings.Contains(csvContent, "飞书应用") {
		t.Error("expected feishu_name '飞书应用' in CSV")
	}
}

// ─── T9: 导出历史记录 ────────────────────────────────────────────────────────

func TestExport_GetHistory(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get history failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["data"]; !ok {
		t.Error("expected 'data' key in history response")
	}
}

// ─── T10: 时间范围无效（start >= end）返回 400 ─────────────────────────────

func TestExport_Preview_InvalidTimeRange(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	body := `{"filter_type":"token_name","token_name":"tok","time_range_start":1714000000,"time_range_end":1712000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/export/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid time range, got %d: %s", w.Code, w.Body.String())
	}
}
