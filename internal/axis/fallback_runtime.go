package axis

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

type FallbackRuntimeMember struct {
	Order            int
	Name             string
	RuntimeGroupName string
	Filter           string
}

func escapeRegexLiteral(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`.`, `\.`,
		`+`, `\+`,
		`*`, `\*`,
		`?`, `\?`,
		`^`, `\^`,
		`$`, `\$`,
		`(`, `\(`,
		`)`, `\)`,
		`[`, `\[`,
		`]`, `\]`,
		`{`, `\{`,
		`}`, `\}`,
		`|`, `\|`,
	)
	return replacer.Replace(value)
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return fmt.Sprintf("%x", sum[:])[:10]
}

func normalizeProxyOrder(proxies []string) []string {
	if len(proxies) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		name := strings.TrimSpace(proxy)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}

func buildFallbackRuntimeMembers(group EgressGroup, availableProxyNames []string) []FallbackRuntimeMember {
	selectedNames := normalizeProxyOrder(group.Proxies)
	if len(selectedNames) == 0 {
		return nil
	}

	availableSet := map[string]struct{}{}
	if len(availableProxyNames) > 0 {
		for _, name := range availableProxyNames {
			availableSet[strings.TrimSpace(name)] = struct{}{}
		}
	}

	members := make([]FallbackRuntimeMember, 0, len(selectedNames))
	for index, name := range selectedNames {
		if len(availableSet) > 0 {
			if _, ok := availableSet[name]; !ok {
				continue
			}
		}
		members = append(members, FallbackRuntimeMember{
			Order:            index + 1,
			Name:             name,
			RuntimeGroupName: fmt.Sprintf("__axis_fb_%s_%02d_%s", shortHash(group.Name), index+1, shortHash(name)),
			Filter:           fmt.Sprintf("^%s$", escapeRegexLiteral(name)),
		})
	}
	return members
}
