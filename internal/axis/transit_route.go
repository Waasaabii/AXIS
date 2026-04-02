package axis

import (
	"fmt"
	"regexp"
	"strings"
)

func sanitizeTransitRuntimeToken(value, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return fallback
	}
	return normalized
}

func buildTransitRuntimeToken(routeName string) string {
	return fmt.Sprintf("%s_%s", sanitizeTransitRuntimeToken(routeName, "route"), shortHash(routeName))
}

func buildTransitUpstreamGroupName(routeName string) string {
	return "__axis_tr_up_" + buildTransitRuntimeToken(routeName)
}

func buildTransitMirrorProviderName(routeName string) string {
	return "__axis_tr_pvd_" + buildTransitRuntimeToken(routeName)
}

func buildTransitMirrorGroupName(routeName string) string {
	return "__axis_tr_grp_" + buildTransitRuntimeToken(routeName)
}

func buildTransitExactFilter(proxyName string) string {
	return fmt.Sprintf("^%s$", regexp.QuoteMeta(strings.TrimSpace(proxyName)))
}

func getListenerRouteMode(listener Listener) string {
	if strings.TrimSpace(listener.RouteMode) == "transit" || (listener.RouteMode == "" && strings.TrimSpace(listener.TransitRoute) != "") {
		return "transit"
	}
	return "direct"
}

func normalizeListenerRouteMode(routeMode, transitRoute string) string {
	if strings.TrimSpace(routeMode) == "transit" || (strings.TrimSpace(routeMode) == "" && strings.TrimSpace(transitRoute) != "") {
		return "transit"
	}
	return "direct"
}

func isTransitRouteEnabled(route TransitRoute) bool {
	return route.Enabled
}
