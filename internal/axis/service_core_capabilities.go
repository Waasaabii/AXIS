package axis

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

type CapabilityLayer struct {
	Protocol    string   `json:"protocol"`
	Import      string   `json:"import"`
	Preserve    string   `json:"preserve"`
	Publish     string   `json:"publish"`
	Usage       string   `json:"usage"`
	Create      string   `json:"create"`
	Diagnostics string   `json:"diagnostics"`
	Warnings    []string `json:"warnings,omitempty"`
}

type CoreCapabilitiesResponse struct {
	CoreName      string            `json:"coreName"`
	CoreVersion   string            `json:"coreVersion,omitempty"`
	CoreDetected  bool              `json:"coreDetected"`
	CheckedAt     string            `json:"checkedAt"`
	Protocols     []CapabilityLayer `json:"protocols"`
	AxisTemplates []string          `json:"axisTemplates"`
	Message       string            `json:"message"`
}

type LLMProposalRequest struct {
	Kind    string         `json:"kind"`
	Goal    string         `json:"goal"`
	Target  string         `json:"target,omitempty"`
	Context map[string]any `json:"context,omitempty"`
	Stream  bool           `json:"stream,omitempty"`
}

type LLMProposalStep struct {
	Title   string `json:"title"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type LLMProposalResponse struct {
	OK           bool              `json:"ok"`
	Enabled      bool              `json:"enabled"`
	Mode         string            `json:"mode"`
	Kind         string            `json:"kind"`
	Status       string            `json:"status"`
	Steps        []LLMProposalStep `json:"steps"`
	Proposal     map[string]any    `json:"proposal"`
	Warnings     []string          `json:"warnings,omitempty"`
	NeedsTest    bool              `json:"needsTest"`
	NeedsConfirm bool              `json:"needsConfirm"`
	Message      string            `json:"message"`
}

func (s *Service) GetCoreCapabilities() CoreCapabilitiesResponse {
	version, detected := detectMihomoVersion(s.config.Runtime.MihomoBinary)
	protocols := []CapabilityLayer{
		{Protocol: "trojan", Import: "支持", Preserve: "支持", Publish: "支持 Mihomo 和 Shadowrocket", Usage: "取决于当前核心出站能力", Create: "支持创建草案和本机监听模板", Diagnostics: "支持 TCP、域名、证书检查"},
		{Protocol: "hysteria2", Import: "支持", Preserve: "支持", Publish: "支持 Mihomo 和 Shadowrocket", Usage: "取决于当前核心出站能力", Create: "支持创建草案和 UDP 端口提示", Diagnostics: "支持 UDP、域名、证书检查"},
		{Protocol: "vmess", Import: "支持", Preserve: "支持 ws/host/path/uuid", Publish: "支持 Mihomo，URI 发布待测试", Usage: "取决于当前核心出站能力", Create: "暂不创建本机节点", Diagnostics: "支持订阅字段风险提示"},
		{Protocol: "vless", Import: "可尝试", Preserve: "支持原始参数保留", Publish: "待测试", Usage: "取决于当前核心出站能力", Create: "暂不创建本机节点", Diagnostics: "支持订阅字段风险提示"},
	}
	message := "已按当前 AXIS 模板展示支持层级"
	if !detected {
		message = "暂未检测到核心版本，协议使用能力以运行时校验为准"
	}
	return CoreCapabilitiesResponse{
		CoreName:      "mihomo",
		CoreVersion:   version,
		CoreDetected:  detected,
		CheckedAt:     time.Now().Format(time.RFC3339),
		Protocols:     protocols,
		AxisTemplates: []string{"parse", "preserve", "publish/mihomo-yaml", "publish/shadowrocket-uri", "runtime/mihomo-outbound", "runtime/listener", "tests"},
		Message:       message,
	}
}

func detectMihomoVersion(binary string) (string, bool) {
	binary = strings.TrimSpace(firstNonEmpty(binary, "mihomo"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "-v").CombinedOutput()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(output)), true
}

func (s *Service) ListLLMModels() (LLMModelsResponse, int) {
	client := newLLMClient(s.config.LLM)
	if !client.enabled() {
		return LLMModelsResponse{OK: false, Enabled: false, Message: "请先填写 LLM 地址和密钥"}, 200
	}
	models, err := client.listModels(context.Background())
	if err != nil {
		return LLMModelsResponse{OK: false, Enabled: true, Message: err.Error()}, 502
	}
	return LLMModelsResponse{OK: true, Enabled: true, Models: models, Message: "模型列表已读取"}, 200
}

func (s *Service) ListLLMModelsWithConfig(request LLMTestRequest) (LLMModelsResponse, int) {
	config := s.config.LLM
	if strings.TrimSpace(request.BaseURL) != "" {
		config.BaseURL = strings.TrimRight(strings.TrimSpace(request.BaseURL), "/")
	}
	if strings.TrimSpace(request.APIKey) != "" {
		config.APIKey = request.APIKey
	}
	config.Enabled = strings.TrimSpace(config.BaseURL) != "" && strings.TrimSpace(config.APIKey) != ""
	client := newLLMClient(config)
	if !client.enabled() {
		return LLMModelsResponse{OK: false, Enabled: false, Message: "请先填写 LLM 地址和密钥"}, 200
	}
	models, err := client.listModels(context.Background())
	if err != nil {
		return LLMModelsResponse{OK: false, Enabled: true, Message: err.Error()}, 502
	}
	return LLMModelsResponse{OK: true, Enabled: true, Models: models, Message: "模型列表已读取"}, 200
}

func (s *Service) TestLLM(request LLMTestRequest) (LLMTestResponse, int) {
	config := s.config.LLM
	if strings.TrimSpace(request.BaseURL) != "" {
		config.BaseURL = strings.TrimRight(strings.TrimSpace(request.BaseURL), "/")
	}
	if strings.TrimSpace(request.APIKey) != "" {
		config.APIKey = request.APIKey
	}
	config.Enabled = strings.TrimSpace(config.BaseURL) != "" && strings.TrimSpace(config.APIKey) != ""
	if strings.TrimSpace(request.Model) != "" {
		config.Model = request.Model
	}
	if strings.TrimSpace(request.Endpoint) != "" {
		config.Endpoint = request.Endpoint
	}
	client := newLLMClient(config)
	if !client.enabled() {
		return LLMTestResponse{OK: false, Endpoint: firstNonEmpty(config.Endpoint, "responses"), Model: config.Model, Message: "请先填写 LLM 地址和密钥"}, 400
	}
	response := client.testModel(context.Background(), config.Model, config.Endpoint)
	if !response.OK {
		return response, 502
	}
	return response, 200
}

func (s *Service) BuildLLMProposal(request LLMProposalRequest) (LLMProposalResponse, int) {
	kind := strings.TrimSpace(request.Kind)
	if kind == "" {
		kind = "rule"
	}
	goal := strings.TrimSpace(request.Goal)
	if goal == "" {
		goal = "根据当前节点生成可测试方案"
	}
	steps := []LLMProposalStep{
		{Title: "整理目标", Status: "done", Message: goal},
		{Title: "读取当前能力", Status: "done", Message: "已使用 AXIS 当前支持层级约束草案"},
	}
	client := newLLMClient(s.config.LLM)
	if client.enabled() && strings.TrimSpace(s.config.LLM.Model) != "" {
		proposal, _, err := client.generateProposal(context.Background(), request)
		if err == nil {
			steps = append(steps, LLMProposalStep{Title: "生成草案", Status: "done", Message: "已由模型生成结构化草案"})
			return LLMProposalResponse{OK: true, Enabled: true, Mode: firstNonEmpty(s.config.LLM.Endpoint, "responses"), Kind: kind, Status: "proposal", Steps: steps, Proposal: proposal, NeedsTest: true, NeedsConfirm: true, Message: "已生成可测试草案，测试通过并确认后才会写入配置。"}, 200
		}
		steps = append(steps, LLMProposalStep{Title: "调用模型", Status: "failed", Message: err.Error()})
	} else {
		steps = append(steps, LLMProposalStep{Title: "调用模型", Status: "skipped", Message: "LLM 未启用或还没有选择模型"})
	}
	steps = append(steps, LLMProposalStep{Title: "生成草案", Status: "done", Message: "已使用 AXIS 确定性草案兜底"})
	proposal := map[string]any{
		"kind":   kind,
		"goal":   goal,
		"status": "proposal",
		"actions": []map[string]any{
			{"type": "test", "title": "先测试连通性", "required": true},
			{"type": "confirm", "title": "测试通过后由用户确认采用", "required": true},
		},
	}
	if kind == "local_node" {
		proposal["nodes"] = []map[string]any{
			{"protocol": "trojan", "listen": "127.0.0.1", "port": 39013, "publish_formats": []string{"mihomo", "shadowrocket"}},
			{"protocol": "hysteria2", "listen": "127.0.0.1", "port": 39014, "publish_formats": []string{"mihomo", "shadowrocket"}},
		}
	}
	return LLMProposalResponse{
		OK:           true,
		Enabled:      false,
		Mode:         "deterministic-draft",
		Kind:         kind,
		Status:       "proposal",
		Steps:        steps,
		Proposal:     proposal,
		Warnings:     []string{"模型不可用时，AXIS 会先生成确定性草案；启用后仍必须测试并确认。"},
		NeedsTest:    true,
		NeedsConfirm: true,
		Message:      "已生成可测试草案，测试通过并确认后才会写入配置。",
	}, 200
}
