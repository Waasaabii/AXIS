package axis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LLMModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	OwnedBy     string `json:"ownedBy,omitempty"`
}

type LLMModelsResponse struct {
	OK      bool           `json:"ok"`
	Enabled bool           `json:"enabled"`
	Models  []LLMModelInfo `json:"models"`
	Message string         `json:"message"`
}

type LLMTestRequest struct {
	BaseURL  string `json:"base_url,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	Model    string `json:"model,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

type LLMTestResponse struct {
	OK       bool   `json:"ok"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
	Content  string `json:"content,omitempty"`
	Message  string `json:"message"`
}

type llmClient struct {
	config LLMConfig
	client *http.Client
}

func newLLMClient(config LLMConfig) *llmClient {
	timeout := max(config.TimeoutSeconds, 60)
	return &llmClient{config: config, client: &http.Client{Timeout: time.Duration(timeout) * time.Second}}
}

func (c *llmClient) enabled() bool {
	return c != nil && c.config.Enabled && strings.TrimSpace(c.config.BaseURL) != "" && strings.TrimSpace(c.config.APIKey) != ""
}

func (c *llmClient) endpointURL(path string) string {
	return strings.TrimRight(strings.TrimSpace(c.config.BaseURL), "/") + path
}

func (c *llmClient) doJSON(ctx context.Context, method, path string, payload any) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpointURL(path), body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if readErr != nil {
		return resp.StatusCode, raw, readErr
	}
	return resp.StatusCode, raw, nil
}

func (c *llmClient) listModels(ctx context.Context) ([]LLMModelInfo, error) {
	if !c.enabled() {
		return nil, errors.New("LLM 尚未配置")
	}
	status, raw, err := c.doJSON(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("模型列表读取失败：%s", summarizeLLMError(raw, status))
	}
	var payload struct {
		Data   []map[string]any `json:"data"`
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	items := payload.Data
	if len(items) == 0 {
		items = payload.Models
	}
	models := make([]LLMModelInfo, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(fmt.Sprint(firstNonNil(item["id"], item["name"])))
		if id == "" || id == "<nil>" {
			continue
		}
		models = append(models, LLMModelInfo{ID: id, Name: id, DisplayName: strings.TrimSpace(fmt.Sprint(item["display_name"])), OwnedBy: strings.TrimSpace(fmt.Sprint(item["owned_by"]))})
	}
	return models, nil
}

func (c *llmClient) generateProposal(ctx context.Context, request LLMProposalRequest) (map[string]any, string, error) {
	if !c.enabled() {
		return nil, "", errors.New("LLM 尚未配置")
	}
	model := strings.TrimSpace(c.config.Model)
	if model == "" {
		return nil, "", errors.New("还没有选择模型")
	}
	prompt := buildLLMProposalPrompt(request)
	endpoint := strings.TrimSpace(c.config.Endpoint)
	if endpoint == "" || endpoint == "responses" {
		proposal, content, err := c.callResponses(ctx, model, prompt)
		if err == nil {
			return proposal, content, nil
		}
		if endpoint == "responses" {
			return nil, "", err
		}
	}
	return c.callChatCompletions(ctx, model, prompt)
}

func (c *llmClient) callResponses(ctx context.Context, model, prompt string) (map[string]any, string, error) {
	payload := map[string]any{"model": model, "input": prompt, "max_output_tokens": 1800}
	status, raw, err := c.doJSON(ctx, http.MethodPost, "/v1/responses", payload)
	if err != nil {
		return nil, "", err
	}
	if status < 200 || status >= 300 {
		return nil, "", fmt.Errorf("Responses 调用失败：%s", summarizeLLMError(raw, status))
	}
	content := extractResponsesText(raw)
	proposal, err := parseProposalJSON(content)
	return proposal, content, err
}

func (c *llmClient) callChatCompletions(ctx context.Context, model, prompt string) (map[string]any, string, error) {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你只输出 JSON，不输出 Markdown，不输出解释。"},
			{"role": "user", "content": prompt},
		},
		"max_completion_tokens": 1800,
	}
	status, raw, err := c.doJSON(ctx, http.MethodPost, "/v1/chat/completions", payload)
	if err != nil {
		return nil, "", err
	}
	if status < 200 || status >= 300 {
		return nil, "", fmt.Errorf("Chat Completions 调用失败：%s", summarizeLLMError(raw, status))
	}
	content := extractChatText(raw)
	proposal, err := parseProposalJSON(content)
	return proposal, content, err
}

func (c *llmClient) testModel(ctx context.Context, model, endpoint string) LLMTestResponse {
	model = strings.TrimSpace(firstNonEmpty(model, c.config.Model))
	endpoint = strings.TrimSpace(firstNonEmpty(endpoint, c.config.Endpoint, "responses"))
	if model == "" {
		return LLMTestResponse{OK: false, Endpoint: endpoint, Model: model, Message: "请先选择模型"}
	}
	prompt := "只输出 JSON：{\"ok\":true,\"message\":\"axis llm test\"}"
	var content string
	var err error
	if endpoint == "chat_completions" || endpoint == "chat" {
		_, content, err = c.callChatCompletions(ctx, model, prompt)
		endpoint = "chat_completions"
	} else {
		_, content, err = c.callResponses(ctx, model, prompt)
		endpoint = "responses"
	}
	if err != nil {
		return LLMTestResponse{OK: false, Endpoint: endpoint, Model: model, Message: err.Error()}
	}
	return LLMTestResponse{OK: true, Endpoint: endpoint, Model: model, Content: content, Message: "模型测试通过"}
}

func buildLLMProposalPrompt(request LLMProposalRequest) string {
	contextJSON, _ := json.Marshal(request.Context)
	return fmt.Sprintf(`请为 AXIS 生成结构化草案。要求：
1. 只能输出 JSON 对象。
2. status 必须是 proposal。
3. 必须包含 actions 数组，每个动作包含 type、title、required。
4. 不允许直接启用规则，不允许要求用户手敲配置。
5. 草案必须可由 AXIS 测试后确认。

类型：%s
目标：%s
上下文：%s
`, firstNonEmpty(request.Kind, "rule"), firstNonEmpty(request.Goal, "根据当前节点生成可测试方案"), string(contextJSON))
}

func extractChatText(raw []byte) string {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Choices) == 0 {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(payload.Choices[0].Message.Content)
}

func extractResponsesText(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return strings.TrimSpace(string(raw))
	}
	if text := strings.TrimSpace(fmt.Sprint(payload["output_text"])); text != "" && text != "<nil>" {
		return text
	}
	outputs, _ := payload["output"].([]any)
	parts := []string{}
	for _, output := range outputs {
		entry, ok := output.(map[string]any)
		if !ok {
			continue
		}
		content, _ := entry["content"].([]any)
		for _, item := range content {
			contentItem, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text := strings.TrimSpace(fmt.Sprint(contentItem["text"])); text != "" && text != "<nil>" {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func parseProposalJSON(content string) (map[string]any, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var proposal map[string]any
	if err := json.Unmarshal([]byte(content), &proposal); err != nil {
		return nil, fmt.Errorf("LLM 没有返回可识别的 JSON 草案")
	}
	if strings.TrimSpace(fmt.Sprint(proposal["status"])) == "" {
		proposal["status"] = "proposal"
	}
	if _, ok := proposal["actions"].([]any); !ok {
		proposal["actions"] = []map[string]any{{"type": "test", "title": "先测试草案", "required": true}, {"type": "confirm", "title": "测试通过后确认采用", "required": true}}
	}
	return proposal, nil
}

func summarizeLLMError(raw []byte, status int) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		if errorValue, ok := payload["error"].(map[string]any); ok {
			if message := strings.TrimSpace(fmt.Sprint(errorValue["message"])); message != "" && message != "<nil>" {
				return message
			}
		}
		if message := strings.TrimSpace(fmt.Sprint(payload["message"])); message != "" && message != "<nil>" {
			return message
		}
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 240 {
		text = text[:240]
	}
	if text == "" {
		text = fmt.Sprintf("HTTP %d", status)
	}
	return text
}
