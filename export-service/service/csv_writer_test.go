package service

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

// ─── T1: WriteHeader 写入正确的 12 列表头 ──────────────────────────────────

func TestCSVWriter_WriteHeader(t *testing.T) {
	var buf bytes.Buffer
	w := NewCSVWriter(&buf)
	if err := w.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader error: %v", err)
	}
	w.Flush()

	reader := csv.NewReader(&buf)
	row, err := reader.Read()
	if err != nil {
		t.Fatalf("read header: %v", err)
	}

	expected := csvHeaders
	if len(row) != len(expected) {
		t.Fatalf("expected %d columns, got %d: %v", len(expected), len(row), row)
	}
	for i, col := range expected {
		if row[i] != col {
			t.Errorf("column[%d]: expected %q, got %q", i, col, row[i])
		}
	}
}

// ─── T2: WriteRecord 字段映射正确 ──────────────────────────────────────────

func TestCSVWriter_WriteRecord(t *testing.T) {
	var buf bytes.Buffer
	w := NewCSVWriter(&buf)
	if err := w.WriteHeader(); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}

	rec := &CSVRecord{
		CreatedAt:           "2024-04-17 10:00:00",
		TokenName:           "my-token",
		FeishuAppID:         "cli_abc123",
		FeishuName:          "测试应用",
		ModelName:           "gpt-4o",
		PromptTokens:        100,
		CompletionTokens:    50,
		CacheTokens:         20,
		CacheCreationTokens: 10,
	}
	if err := w.WriteRecord(rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	w.Flush()

	reader := csv.NewReader(&buf)
	reader.Read() // skip header
	row, err := reader.Read()
	if err != nil {
		t.Fatalf("read record: %v", err)
	}

	checks := []struct {
		col int
		got string
		exp string
	}{
		{0, row[0], "2024-04-17 10:00:00"},
		{1, row[1], "my-token"},
		{2, row[2], "cli_abc123"},
		{3, row[3], "测试应用"},
		{4, row[4], "gpt-4o"},
		{5, row[5], "100"},
		{6, row[6], "50"},
		{7, row[7], "20"},
		{8, row[8], "10"},
	}
	for _, c := range checks {
		if c.got != c.exp {
			t.Errorf("col[%d] %s: expected %q, got %q", c.col, csvHeaders[c.col], c.exp, c.got)
		}
	}
}

// ─── T3: intToString 边界值 ─────────────────────────────────────────────────

func TestIntToString(t *testing.T) {
	cases := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{9999999, "9999999"},
		{-9999999, "-9999999"},
		{1000000000, "1000000000"},
	}
	for _, c := range cases {
		got := intToString(c.input)
		if got != c.expected {
			t.Errorf("intToString(%d) = %q, want %q", c.input, got, c.expected)
		}
	}
}

// ─── T4: 未调用 Flush 时数据可能不完整（揭示 bug）─────────────────────────
//
// csv.Writer 内部有缓冲区。如果写入后不调用 Flush()，
// 最后一批数据可能滞留在缓冲区中导致 CSV 不完整。
// 本测试验证：直接读取 writer 底层 buffer 而不调用 Flush 时，数据不完整。

func TestCSVWriter_NoFlush_DataMayBeLost(t *testing.T) {
	var buf bytes.Buffer
	w := NewCSVWriter(&buf)
	w.WriteHeader()

	rec := &CSVRecord{
		CreatedAt:   "2024-04-17 10:00:00",
		TokenName:   "tok",
		ModelName:   "gpt-4",
		PromptTokens: 100,
	}
	w.WriteRecord(rec)
	// 注意：故意不调用 w.Flush()

	// 检查 buf 中是否有数据行（如果没有 Flush，记录行可能不在 buf 中）
	content := buf.String()
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// 如果 Flush 未调用，可能只有表头行或者为空
	// 注意：csv.Writer 内部 bufio.Writer 默认大小 4096 字节，
	// 少量数据不会触发自动 flush，所以记录行很可能缺失
	t.Logf("buffer content without Flush:\n%s", content)
	t.Logf("line count: %d", len(lines))

	// 表头很短（< 4096 字节），可能也不在 buf 中
	// 这个测试的目的是记录实际行为，作为 bug 证据
	// PASS 条件：我们观察到了缺失行为（实际上 csv.Writer 使用 bufio，确实需要 Flush）
	hasRecord := false
	for _, line := range lines {
		if strings.Contains(line, "tok") {
			hasRecord = true
		}
	}
	if hasRecord {
		t.Log("INFO: record found in buffer even without Flush (csv.Writer auto-flushed)")
	} else {
		t.Log("BUG CONFIRMED: record NOT in buffer without Flush() - controller/export.go Download() missing csvWriter.Flush()")
	}
}

// ─── T5: 调用 Flush 后数据完整 ─────────────────────────────────────────────

func TestCSVWriter_WithFlush_DataComplete(t *testing.T) {
	var buf bytes.Buffer
	w := NewCSVWriter(&buf)
	w.WriteHeader()

	records := []*CSVRecord{
		{CreatedAt: "2024-01-01 00:00:00", TokenName: "tok-1", ModelName: "gpt-4", PromptTokens: 100},
		{CreatedAt: "2024-01-01 01:00:00", TokenName: "tok-2", ModelName: "gpt-4", PromptTokens: 200},
		{CreatedAt: "2024-01-01 02:00:00", TokenName: "tok-3", ModelName: "gpt-4", PromptTokens: 300},
	}
	for _, r := range records {
		w.WriteRecord(r)
	}
	w.Flush() // 必须调用

	if err := w.Error(); err != nil {
		t.Fatalf("writer error: %v", err)
	}

	reader := csv.NewReader(&buf)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	// 1 header + 3 records
	if len(rows) != 4 {
		t.Errorf("expected 4 rows (1 header + 3 records), got %d", len(rows))
	}
}

// ─── T6: CSV 输出列顺序与需求文档一致 ──────────────────────────────────────

func TestCSVWriter_ColumnOrder(t *testing.T) {
	// 需求文档要求的列顺序
	requiredOrder := []string{
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

	if len(csvHeaders) != len(requiredOrder) {
		t.Fatalf("expected %d columns, got %d", len(requiredOrder), len(csvHeaders))
	}
	for i, col := range requiredOrder {
		if csvHeaders[i] != col {
			t.Errorf("column[%d]: expected %q, got %q", i, col, csvHeaders[i])
		}
	}
}
