package axis

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type LocalNodeCheckItem struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

type LocalNodeCheckResponse struct {
	OK      bool                 `json:"ok"`
	Name    string               `json:"name"`
	Checks  []LocalNodeCheckItem `json:"checks"`
	Message string               `json:"message"`
}

type LocalNodeConnection struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Address  string `json:"address"`
	URL      string `json:"url"`
}

type LocalNodeConnectionResponse struct {
	OK          bool                  `json:"ok"`
	Name        string                `json:"name"`
	Protocol    string                `json:"protocol"`
	Host        string                `json:"host"`
	Port        int                   `json:"port"`
	Connections []LocalNodeConnection `json:"connections"`
	Warnings    []string              `json:"warnings,omitempty"`
}

type BaotaConfigResponse struct {
	OK             bool     `json:"ok"`
	Name           string   `json:"name"`
	Protocol       string   `json:"protocol"`
	Mode           string   `json:"mode"`
	Target         string   `json:"target"`
	Snippet        string   `json:"snippet"`
	Steps          []string `json:"steps"`
	Warnings       []string `json:"warnings,omitempty"`
	ApplySupported bool     `json:"applySupported"`
}

func (s *Service) CheckLocalNode(name string) (LocalNodeCheckResponse, int) {
	source, ok := s.findLocalNode(name)
	if !ok {
		return LocalNodeCheckResponse{OK: false, Name: name, Message: "没有找到这个本机节点"}, 404
	}
	checks := []LocalNodeCheckItem{}
	checks = append(checks, checkExternalHost(source.LocalNode.ExternalHost))
	checks = append(checks, checkLocalListen(source))
	checks = append(checks, checkCertificateFiles(source))
	checks = append(checks, checkBaotaMode(source))
	ok = true
	for _, item := range checks {
		if !item.OK {
			ok = false
			break
		}
	}
	message := "本机节点检查通过，可以复制连接信息"
	if !ok {
		message = "本机节点还需要处理检查项后再使用"
	}
	return LocalNodeCheckResponse{OK: ok, Name: source.Name, Checks: checks, Message: message}, 200
}

func (s *Service) GetLocalNodeConnection(name string) (LocalNodeConnectionResponse, int) {
	source, ok := s.findLocalNode(name)
	if !ok {
		return LocalNodeConnectionResponse{OK: false, Name: name, Warnings: []string{"没有找到这个本机节点"}}, 404
	}
	if strings.TrimSpace(source.LocalNode.ExternalHost) == "" {
		return LocalNodeConnectionResponse{OK: false, Name: source.Name, Protocol: source.Protocol, Warnings: []string{"请先填写对外域名，AXIS 不生成 IP 直连信息"}}, 400
	}
	port := source.LocalNode.ExternalPort
	if port == 0 {
		port = defaultLocalNodeExternalPort(source.Protocol, source.LocalNode.Port)
	}
	connections := make([]LocalNodeConnection, 0, len(source.LocalNode.Users))
	for _, user := range source.LocalNode.Users {
		connections = append(connections, LocalNodeConnection{Username: user.Username, Password: user.Password, Address: fmt.Sprintf("%s:%d", source.LocalNode.ExternalHost, port), URL: buildLocalNodeURL(source, user, port)})
	}
	warnings := []string{}
	if source.Protocol == "hysteria2" {
		warnings = append(warnings, "Hysteria2 需要 UDP 可达，普通 HTTPS 反向代理不能使用")
	}
	if source.Protocol == "trojan" {
		warnings = append(warnings, "Trojan 需要可用证书和 TCP 转发，不适合普通网页反代")
	}
	return LocalNodeConnectionResponse{OK: true, Name: source.Name, Protocol: normalizeLocalNodeListenerType(source.Protocol), Host: source.LocalNode.ExternalHost, Port: port, Connections: connections, Warnings: warnings}, 200
}

func (s *Service) GetLocalNodeBaotaConfig(name string) (BaotaConfigResponse, int) {
	source, ok := s.findLocalNode(name)
	if !ok {
		return BaotaConfigResponse{OK: false, Name: name, Warnings: []string{"没有找到这个本机节点"}}, 404
	}
	protocol := normalizeLocalNodeListenerType(source.Protocol)
	target := fmt.Sprintf("%s:%d", firstNonEmpty(source.LocalNode.Listen, "127.0.0.1"), source.LocalNode.Port)
	response := BaotaConfigResponse{OK: true, Name: source.Name, Protocol: protocol, Mode: source.LocalNode.AccessMode, Target: target, ApplySupported: false}
	switch protocol {
	case "http", "socks", "trojan":
		listenPort := source.LocalNode.ExternalPort
		if listenPort == 0 {
			listenPort = defaultLocalNodeExternalPort(protocol, source.LocalNode.Port)
		}
		response.Snippet = fmt.Sprintf("server {\n    listen %d;\n    proxy_pass %s;\n}\n", listenPort, target)
		response.Steps = []string{"在宝塔确认 nginx 已开启 TCP 转发能力", "把这段配置作为 AXIS 专用 TCP 配置应用", "保存后测试 nginx 配置，再重载 nginx"}
		if protocol == "trojan" {
			response.Warnings = append(response.Warnings, "Trojan 如需共用 443，需要确认不会和现有网站冲突")
		}
	case "hysteria2":
		listenPort := source.LocalNode.ExternalPort
		if listenPort == 0 {
			listenPort = 39014
		}
		response.Snippet = fmt.Sprintf("server {\n    listen %d udp;\n    proxy_pass %s;\n}\n", listenPort, target)
		response.Steps = []string{"在云安全组和系统防火墙放行 UDP 端口", "把这段配置作为 AXIS 专用 UDP 配置应用", "保存后测试 nginx 配置，再重载 nginx"}
		response.Warnings = append(response.Warnings, "Hysteria2 不能走普通网页反向代理")
	default:
		response.OK = false
		response.Warnings = append(response.Warnings, "当前协议暂不支持生成宝塔配置")
	}
	return response, 200
}

func (s *Service) findLocalNode(name string) (NodeSource, bool) {
	decodedName, _ := url.PathUnescape(name)
	for _, source := range s.config.NodeSources {
		if source.Name == decodedName && source.Type == "local_node" {
			return source, true
		}
	}
	return NodeSource{}, false
}

func checkExternalHost(host string) LocalNodeCheckItem {
	if strings.TrimSpace(host) == "" {
		return LocalNodeCheckItem{Name: "访问域名", OK: false, Message: "还没有填写对外域名", Action: "填写已经解析到这台服务器的域名"}
	}
	_, err := net.LookupHost(host)
	if err != nil {
		return LocalNodeCheckItem{Name: "访问域名", OK: false, Message: "域名暂时无法解析", Action: "检查 DNS 解析是否生效"}
	}
	return LocalNodeCheckItem{Name: "访问域名", OK: true, Message: "域名可以解析"}
}

func checkLocalListen(source NodeSource) LocalNodeCheckItem {
	address := fmt.Sprintf("%s:%d", firstNonEmpty(source.LocalNode.Listen, "127.0.0.1"), source.LocalNode.Port)
	if source.Protocol == "hysteria2" {
		return LocalNodeCheckItem{Name: "本机服务", OK: true, Message: fmt.Sprintf("将使用 UDP 本机端口 %s", address), Action: "创建后请确认 UDP 端口已放行"}
	}
	conn, err := net.DialTimeout("tcp", address, 800*time.Millisecond)
	if err != nil {
		return LocalNodeCheckItem{Name: "本机服务", OK: false, Message: "本机端口暂未连通", Action: "应用运行配置后再检查"}
	}
	_ = conn.Close()
	return LocalNodeCheckItem{Name: "本机服务", OK: true, Message: "本机端口可以连接"}
}

func checkCertificateFiles(source NodeSource) LocalNodeCheckItem {
	if source.Protocol != "trojan" && source.Protocol != "hysteria2" {
		return LocalNodeCheckItem{Name: "证书", OK: true, Message: "当前协议不需要在 AXIS 里配置证书"}
	}
	if source.LocalNode.AccessMode == "bt_reverse_proxy" && source.LocalNode.Certificate == "" && source.LocalNode.PrivateKey == "" {
		return LocalNodeCheckItem{Name: "证书", OK: true, Message: "证书可由宝塔管理"}
	}
	if source.LocalNode.Certificate == "" || source.LocalNode.PrivateKey == "" {
		return LocalNodeCheckItem{Name: "证书", OK: false, Message: "证书文件还不完整", Action: "填写证书和私钥文件路径，或改为由宝塔处理"}
	}
	if _, err := os.Stat(source.LocalNode.Certificate); err != nil {
		return LocalNodeCheckItem{Name: "证书", OK: false, Message: "证书文件不存在", Action: "检查证书文件路径"}
	}
	if _, err := os.Stat(source.LocalNode.PrivateKey); err != nil {
		return LocalNodeCheckItem{Name: "证书", OK: false, Message: "私钥文件不存在", Action: "检查私钥文件路径"}
	}
	return LocalNodeCheckItem{Name: "证书", OK: true, Message: "证书文件存在"}
}

func checkBaotaMode(source NodeSource) LocalNodeCheckItem {
	switch normalizeLocalNodeListenerType(source.Protocol) {
	case "http":
		return LocalNodeCheckItem{Name: "宝塔设置", OK: true, Message: "HTTP 可以使用 TCP 转发或直连端口"}
	case "socks":
		return LocalNodeCheckItem{Name: "宝塔设置", OK: true, Message: "SOCKS 需要 TCP 转发或直连端口"}
	case "trojan":
		return LocalNodeCheckItem{Name: "宝塔设置", OK: true, Message: "Trojan 需要 TCP 转发，不能用普通网页反代"}
	case "hysteria2":
		return LocalNodeCheckItem{Name: "宝塔设置", OK: true, Message: "Hysteria2 需要 UDP 转发或 UDP 直连"}
	default:
		return LocalNodeCheckItem{Name: "宝塔设置", OK: false, Message: "当前协议暂不支持"}
	}
}

func buildLocalNodeURL(source NodeSource, user ListenerUser, port int) string {
	host := source.LocalNode.ExternalHost
	name := url.QueryEscape(source.Name)
	switch normalizeLocalNodeListenerType(source.Protocol) {
	case "http":
		return (&url.URL{Scheme: "http", User: url.UserPassword(user.Username, user.Password), Host: fmt.Sprintf("%s:%d", host, port)}).String()
	case "socks":
		return (&url.URL{Scheme: "socks5", User: url.UserPassword(user.Username, user.Password), Host: fmt.Sprintf("%s:%d", host, port)}).String()
	case "trojan":
		values := url.Values{}
		values.Set("security", "tls")
		if firstNonEmpty(source.LocalNode.SNI, source.LocalNode.ExternalHost) != "" {
			values.Set("sni", firstNonEmpty(source.LocalNode.SNI, source.LocalNode.ExternalHost))
		}
		return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", url.QueryEscape(user.Password), host, port, values.Encode(), name)
	case "hysteria2":
		values := url.Values{}
		if firstNonEmpty(source.LocalNode.SNI, source.LocalNode.ExternalHost) != "" {
			values.Set("sni", firstNonEmpty(source.LocalNode.SNI, source.LocalNode.ExternalHost))
		}
		return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s", url.QueryEscape(user.Password), host, port, values.Encode(), name)
	default:
		return fmt.Sprintf("%s:%d", host, port)
	}
}
