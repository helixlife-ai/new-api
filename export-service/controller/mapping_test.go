package controller_test

import (
	"bytes"
	"encoding/json"
	"export-service/controller"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── T1: 创建映射 ────────────────────────────────────────────────────────────

func TestMapping_Create(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	body := `{"feishuAppId":"cli_abc","feishuName":"测试应用","tokenId":100,"tokenName":"test-token","tags":["batch-1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp controller.MappingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.FeishuAppID != "cli_abc" {
		t.Errorf("feishuAppId mismatch: %s", resp.FeishuAppID)
	}
	if resp.TokenName != "test-token" {
		t.Errorf("tokenName mismatch: %s", resp.TokenName)
	}
	if len(resp.Tags) != 1 || resp.Tags[0] != "batch-1" {
		t.Errorf("tags mismatch: %v", resp.Tags)
	}
}

// ─── T2: 获取映射列表 ────────────────────────────────────────────────────────

func TestMapping_List(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{"feishuAppId":"cli_%d","feishuName":"App%d","tokenId":%d,"tokenName":"tok-%d"}`, i, i, i+100, i)
		req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %s", w.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/mappings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d %s", w.Code, w.Body.String())
	}
	var resp []*controller.MappingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 mappings, got %d", len(resp))
	}
}

// ─── T3: 更新映射 ────────────────────────────────────────────────────────────

func TestMapping_Update(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	createBody := `{"feishuAppId":"cli_upd","feishuName":"Old Name","tokenId":200,"tokenName":"old-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var created controller.MappingResponse
	json.Unmarshal(w.Body.Bytes(), &created)

	updateBody := `{"feishuName":"New Name","tags":["updated"]}`
	req2 := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/mappings/%d", created.ID), strings.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("update failed: %d %s", w2.Code, w2.Body.String())
	}
	var updated controller.MappingResponse
	json.Unmarshal(w2.Body.Bytes(), &updated)
	if updated.FeishuName != "New Name" {
		t.Errorf("feishuName not updated: %s", updated.FeishuName)
	}
}

// ─── T4: 删除单条映射 ────────────────────────────────────────────────────────

func TestMapping_DeleteSingle(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	createBody := `{"feishuAppId":"cli_del","feishuName":"Del App","tokenId":300,"tokenName":"del-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var created controller.MappingResponse
	json.Unmarshal(w.Body.Bytes(), &created)

	req2 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/mappings/%d", created.ID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("delete failed: %d %s", w2.Code, w2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/mappings", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var list []*controller.MappingResponse
	json.Unmarshal(w3.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(list))
	}
}

// ─── T5: 按标签批量删除 ──────────────────────────────────────────────────────

func TestMapping_BatchDeleteByTag(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	creates := []string{
		`{"feishuAppId":"cli_b1a","feishuName":"B1A","tokenId":401,"tokenName":"b1a","tags":["batch-1"]}`,
		`{"feishuAppId":"cli_b1b","feishuName":"B1B","tokenId":402,"tokenName":"b1b","tags":["batch-1"]}`,
		`{"feishuAppId":"cli_b2","feishuName":"B2","tokenId":403,"tokenName":"b2","tags":["batch-2"]}`,
	}
	for _, b := range creates {
		req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %s", w.Body.String())
		}
	}

	delBody := `{"tags":["batch-1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/mappings/batch-delete", strings.NewReader(delBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("batch-delete failed: %d %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/mappings", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var list []*controller.MappingResponse
	json.Unmarshal(w2.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 remaining (batch-2), got %d", len(list))
	}
	if len(list) == 1 && (len(list[0].Tags) == 0 || list[0].Tags[0] != "batch-2") {
		t.Errorf("expected remaining to be batch-2, got tags: %v", list[0].Tags)
	}
}

// ─── T6: CSV 导入——labels 参数被忽略（BUG 复现）────────────────────────────

func TestMapping_Import_LabelsParamIgnored_Bug(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	csvContent := "feishu_app_id,feishu_name,token_id,token_name\ncli_imp1,Import App,500,imp-tok-1\n"

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("labels", "batch-2026-04-16")
	fw, _ := mw.CreateFormFile("file", "test.csv")
	fw.Write([]byte(csvContent))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/mappings/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("import failed: %d %s", w.Code, w.Body.String())
	}

	var importResp controller.ImportMappingsResponse
	json.Unmarshal(w.Body.Bytes(), &importResp)
	if importResp.ImportedCount != 1 {
		t.Fatalf("expected 1 imported, got %d", importResp.ImportedCount)
	}

	// 验证导入记录的 labels
	req2 := httptest.NewRequest(http.MethodGet, "/api/mappings", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var list []*controller.MappingResponse
	json.Unmarshal(w2.Body.Bytes(), &list)

	if len(list) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(list))
	}

	hasExpectedLabel := false
	for _, tag := range list[0].Tags {
		if tag == "batch-2026-04-16" {
			hasExpectedLabel = true
			break
		}
	}

	if hasExpectedLabel {
		t.Log("PASS: labels 参数已正确应用到导入记录")
	} else {
		t.Errorf("BUG CONFIRMED: POST /api/mappings/import 的 labels 参数被忽略，导入记录 tags=%v（期望包含 'batch-2026-04-16'）", list[0].Tags)
	}
}

// ─── T7: 标签筛选列表 ────────────────────────────────────────────────────────

func TestMapping_ListByLabel(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	creates := []string{
		`{"feishuAppId":"cli_la1","feishuName":"LA1","tokenId":601,"tokenName":"la1","tags":["april"]}`,
		`{"feishuAppId":"cli_la2","feishuName":"LA2","tokenId":602,"tokenName":"la2","tags":["may"]}`,
	}
	for _, b := range creates {
		req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/mappings?label=april", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list by label failed: %d", w.Code)
	}
	var list []*controller.MappingResponse
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 mapping with label 'april', got %d", len(list))
	}
}

// ─── T8: 重复 feishu_app_id 返回 409 ────────────────────────────────────────

func TestMapping_Create_DuplicateFeishuAppID(t *testing.T) {
	setupTestDB(t)
	r := newTestRouter(t, "http://localhost:3000")

	body := `{"feishuAppId":"cli_dup","feishuName":"Dup App","tokenId":700,"tokenName":"dup-tok"}`
	req := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create failed: %d %s", w.Code, w.Body.String())
	}

	body2 := `{"feishuAppId":"cli_dup","feishuName":"Dup App2","tokenId":701,"tokenName":"dup-tok-2"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/mappings", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate feishu_app_id, got %d: %s", w2.Code, w2.Body.String())
	}
}
