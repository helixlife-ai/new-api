package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const streamPageSize = 100

// NewAPIClient newapi-tools 客户端
type NewAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

// LogEntry 日志条目（对应 newapi-tools /api/logs 返回格式）
type LogEntry struct {
	ID                  int    `json:"id"`
	TokenName           string `json:"token_name"`
	ModelName           string `json:"model_name"`
	PromptTokens        int    `json:"prompt_tokens"`
	CompletionTokens    int    `json:"completion_tokens"`
	CacheTokens         int    `json:"cache_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	TotalTokens         int    `json:"total_tokens"`
	Quota               int    `json:"quota"`
	UseTime             int    `json:"use_time"`
	CreatedAt           int64  `json:"created_at"`
}

// logsPageResponse /api/logs 分页响应
type logsPageResponse struct {
	Data struct {
		Items    []LogEntry `json:"items"`
		Total    int64      `json:"total"`
		Page     int        `json:"page"`
		PageSize int        `json:"page_size"`
	} `json:"data"`
}

// NewNewAPIClient 创建 newapi-tools 客户端
func NewNewAPIClient(baseURL string) *NewAPIClient {
	return &NewAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EstimateLogCount 估算日志数量（单个 token 名称）
func (c *NewAPIClient) EstimateLogCount(ctx context.Context, tokenNames []string, startTime, endTime int64) (int, error) {
	if len(tokenNames) == 0 {
		return 0, nil
	}
	total := int64(0)
	for _, name := range tokenNames {
		resp, err := c.fetchPage(ctx, name, startTime, endTime, 1, 1)
		if err != nil {
			return int(total) + len(tokenNames)*100, nil // 粗略估计
		}
		total += resp.Data.Total
	}
	return int(total), nil
}

// StreamLogsByTokenName 流式获取日志（按单个 token 名称）
func (c *NewAPIClient) StreamLogsByTokenName(ctx context.Context, tokenName string, startTime, endTime int64, callback func(*LogEntry) error) error {
	return c.streamPages(ctx, tokenName, startTime, endTime, callback)
}

// StreamLogsByTokenNames 流式获取日志（按多个 token 名称，逐个查询）
func (c *NewAPIClient) StreamLogsByTokenNames(ctx context.Context, tokenNames []string, startTime, endTime int64, callback func(*LogEntry) error) error {
	for _, name := range tokenNames {
		if err := c.streamPages(ctx, name, startTime, endTime, callback); err != nil {
			return err
		}
	}
	return nil
}

// streamPages 翻页拉取所有数据并逐条调用 callback
func (c *NewAPIClient) streamPages(ctx context.Context, tokenName string, startTime, endTime int64, callback func(*LogEntry) error) error {
	page := 1
	for {
		resp, err := c.fetchPage(ctx, tokenName, startTime, endTime, page, streamPageSize)
		if err != nil {
			return err
		}

		for i := range resp.Data.Items {
			if err := callback(&resp.Data.Items[i]); err != nil {
				return err
			}
		}

		if len(resp.Data.Items) < streamPageSize {
			break
		}
		page++
	}
	return nil
}

// fetchPage 调用 newapi-tools GET /api/logs 取一页数据
func (c *NewAPIClient) fetchPage(ctx context.Context, tokenName string, startTime, endTime int64, page, pageSize int) (*logsPageResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/logs")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if tokenName != "" {
		q.Set("token_name", tokenName)
	}
	q.Set("start_timestamp", strconv.FormatInt(startTime, 10))
	q.Set("end_timestamp", strconv.FormatInt(endTime, 10))
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("newapi-tools /api/logs returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result logsPageResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
