package axis

import (
	"fmt"
	"regexp"
	"strings"
)

func matchesGroup(group EgressGroup, node NodeInfo) bool {
	include := true
	if group.Filter != "" {
		regex, err := regexp.Compile(group.Filter)
		if err == nil {
			include = regex.MatchString(node.Name)
		}
	}
	exclude := false
	if group.ExcludeFilter != "" {
		regex, err := regexp.Compile(group.ExcludeFilter)
		if err == nil {
			exclude = regex.MatchString(node.Name)
		}
	}
	return include && !exclude
}

func relaySourceGroupName(groupName string) string {
	return groupName + "::source"
}

func landingProxyRuntimeName(name string) string {
	return "landing::" + name
}

func formatRouteSummary(groupName, currentProxy, landingProxy string) string {
	base := firstNonEmpty(currentProxy, groupName)
	if landingProxy == "" {
		return fmt.Sprintf("%s -> 公网", base)
	}
	return fmt.Sprintf("%s -> %s", base, landingProxy)
}

func formatTransitRouteSummary(upstreamProxy string, targetSummary string, egressGroup string) string {
	nextHop := firstNonEmpty(targetSummary, egressGroup)
	if nextHop == "" {
		nextHop = "公网"
	}
	if strings.TrimSpace(upstreamProxy) == "" {
		return nextHop
	}
	return fmt.Sprintf("%s -> %s", upstreamProxy, nextHop)
}

func buildTransitRouteBindingError(routeView *TransitRouteView) string {
	if routeView == nil {
		return "中转线路不存在"
	}
	if routeView.Status == "disabled" {
		return fmt.Sprintf("中转线路 %s 已停用", routeView.Name)
	}
	if routeView.ProviderMissing {
		return fmt.Sprintf("中转线路 %s 的来源已删除", routeView.Name)
	}
	if routeView.ProviderDisabled {
		return fmt.Sprintf("中转线路 %s 的来源已停用", routeView.Name)
	}
	if routeView.TransitProxyMissing {
		return fmt.Sprintf("中转线路 %s 当前不存在中转节点 %s", routeView.Name, routeView.UpstreamProxyName)
	}
	if routeView.EgressGroupMissing {
		return fmt.Sprintf("中转线路 %s 的落地出口组已删除", routeView.Name)
	}
	if routeView.EgressProviderMissing {
		return fmt.Sprintf("中转线路 %s 的落地出口 provider 已删除", routeView.Name)
	}
	if routeView.EgressProviderDisabled {
		return fmt.Sprintf("中转线路 %s 的落地出口 provider 已停用", routeView.Name)
	}
	if routeView.LandingMissing {
		return fmt.Sprintf("中转线路 %s 绑定的落地节点已删除", routeView.Name)
	}
	if routeView.LandingDisabled {
		return fmt.Sprintf("中转线路 %s 绑定的落地节点已停用", routeView.Name)
	}
	return fmt.Sprintf("中转线路 %s 当前不可用", routeView.Name)
}

func buildFallbackCandidates(group EgressGroup, nodes []NodeInfo) []GroupCandidate {
	nodeMap := map[string]NodeInfo{}
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	candidates := make([]GroupCandidate, 0, len(group.Proxies))
	for _, proxyName := range normalizeProxyOrder(group.Proxies) {
		node, ok := nodeMap[proxyName]
		if !ok {
			continue
		}
		candidateID := node.Name
		candidateName := node.Name
		if group.LandingProxy != "" {
			candidateID = fmt.Sprintf("%s -> %s", node.Name, group.LandingProxy)
			candidateName = candidateID
		}
		candidates = append(candidates, GroupCandidate{
			ID:       candidateID,
			Name:     candidateName,
			Type:     node.Type,
			Server:   node.Server,
			Port:     node.Port,
			NodeName: node.Name,
		})
	}
	return candidates
}

func (s *Service) buildTransitRouteView(route TransitRoute, groupsByName map[string]GroupView, providerNames map[string]struct{}, enabledProviders map[string]struct{}) TransitRouteView {
	providerRecord := s.getProviderRecord(route.UpstreamProvider)
	transitProxyKnown := len(providerRecord.Nodes) > 0
	transitProxyMissing := false
	if transitProxyKnown {
		transitProxyMissing = true
		for _, node := range providerRecord.Nodes {
			if node.Name == route.UpstreamProxyName {
				transitProxyMissing = false
				break
			}
		}
	}

	targetGroup, egressGroupExists := groupsByName[route.EgressGroup]
	_, providerExists := providerNames[route.UpstreamProvider]
	_, providerEnabled := enabledProviders[route.UpstreamProvider]
	routeState := s.state.TransitRoutes[route.Name]

	status := "configured"
	if !route.Enabled {
		status = "disabled"
	} else if !providerExists || !egressGroupExists {
		status = "orphaned"
	} else if !providerEnabled || transitProxyMissing || targetGroup.ProviderMissing || targetGroup.ProviderDisabled || targetGroup.LandingMissing || targetGroup.LandingDisabled {
		status = "degraded"
	}

	return TransitRouteView{
		Name:                   route.Name,
		Enabled:                route.Enabled,
		UpstreamProvider:       route.UpstreamProvider,
		UpstreamProxyName:      route.UpstreamProxyName,
		EgressGroup:            route.EgressGroup,
		Notes:                  route.Notes,
		CurrentProxy:           targetGroup.Current,
		CandidateCount:         targetGroup.CandidateCount,
		EgressGroupMode:        targetGroup.Mode,
		RuntimeGroupName:       buildTransitMirrorGroupName(route.Name),
		RouteSummary:           formatTransitRouteSummary(route.UpstreamProxyName, targetGroup.RouteSummary, route.EgressGroup),
		LastTestedAt:           routeState.LastTestedAt,
		LastTestStatus:         routeState.LastTestStatus,
		LastTestMessage:        routeState.LastTestMessage,
		LastTestDelay:          routeState.LastTestDelay,
		LastTestURL:            routeState.LastTestURL,
		Status:                 status,
		ProviderMissing:        !providerExists,
		ProviderDisabled:       providerExists && !providerEnabled,
		TransitProxyMissing:    transitProxyMissing,
		EgressGroupMissing:     !egressGroupExists,
		EgressProviderMissing:  targetGroup.ProviderMissing,
		EgressProviderDisabled: targetGroup.ProviderDisabled,
		LandingMissing:         targetGroup.LandingMissing,
		LandingDisabled:        targetGroup.LandingDisabled,
	}
}

func (s *Service) buildGroupView(group EgressGroup) GroupView {
	providerRecord := s.getProviderRecord(group.Provider)
	candidates := []GroupCandidate{}
	if group.Mode == "fallback" {
		candidates = buildFallbackCandidates(group, providerRecord.Nodes)
	} else {
		for _, node := range providerRecord.Nodes {
			if matchesGroup(group, node) {
				candidateID := node.Name
				candidateName := node.Name
				if group.LandingProxy != "" {
					candidateID = fmt.Sprintf("%s -> %s", node.Name, group.LandingProxy)
					candidateName = candidateID
				}
				candidates = append(candidates, GroupCandidate{ID: candidateID, Name: candidateName, Type: node.Type, Server: node.Server, Port: node.Port, NodeName: node.Name})
			}
		}
	}
	currentValue := s.state.GroupSelections[group.Name]
	currentLabel := ""
	for _, candidate := range candidates {
		if candidate.ID == currentValue || (group.LandingProxy != "" && candidate.NodeName == currentValue) {
			currentValue = candidate.ID
			currentLabel = candidate.Name
			break
		}
	}
	if currentValue == "" && len(candidates) > 0 {
		currentValue = candidates[0].ID
		currentLabel = candidates[0].Name
	}
	if currentLabel == "" && len(candidates) > 0 {
		currentLabel = candidates[0].Name
	}
	groupState := s.state.Groups[group.Name]
	return GroupView{
		Name:                group.Name,
		Mode:                firstNonEmpty(group.Mode, "manual"),
		Provider:            group.Provider,
		Filter:              group.Filter,
		ExcludeFilter:       group.ExcludeFilter,
		CandidateCount:      len(candidates),
		Current:             currentLabel,
		CurrentValue:        currentValue,
		Candidates:          candidates,
		ProxyOrder:          normalizeProxyOrder(group.Proxies),
		LastHealthcheckAt:   groupState.LastHealthcheckAt,
		LandingProxy:        group.LandingProxy,
		HealthCheckURL:      group.HealthCheckURL,
		HealthCheckInterval: group.Interval,
		RouteSummary:        formatRouteSummary(group.Name, currentLabel, group.LandingProxy),
	}
}

func (s *Service) GetGroups() []GroupView {
	providerNames := map[string]struct{}{}
	enabledProviders := map[string]struct{}{}
	for _, subscription := range s.config.Subscriptions {
		providerNames[subscription.Name] = struct{}{}
		if subscription.Enabled {
			enabledProviders[subscription.Name] = struct{}{}
		}
	}
	landingNames := map[string]struct{}{}
	enabledLandings := map[string]struct{}{}
	for _, landing := range s.config.LandingProxies {
		landingNames[landing.Name] = struct{}{}
		if landing.Enabled {
			enabledLandings[landing.Name] = struct{}{}
		}
	}

	views := make([]GroupView, 0, len(s.config.EgressGroups))
	for _, group := range s.config.EgressGroups {
		view := s.buildGroupView(group)
		_, providerExists := providerNames[group.Provider]
		_, providerEnabled := enabledProviders[group.Provider]
		_, landingExists := landingNames[group.LandingProxy]
		_, landingEnabled := enabledLandings[group.LandingProxy]
		view.ProviderMissing = !providerExists
		view.ProviderDisabled = providerExists && !providerEnabled
		view.LandingMissing = group.LandingProxy != "" && !landingExists
		view.LandingDisabled = group.LandingProxy != "" && landingExists && !landingEnabled
		views = append(views, view)
	}
	return views
}

func (s *Service) GetTransitRoutes() []TransitRouteView {
	groupsByName := map[string]GroupView{}
	for _, group := range s.GetGroups() {
		groupsByName[group.Name] = group
	}

	providerNames := map[string]struct{}{}
	enabledProviders := map[string]struct{}{}
	for _, subscription := range s.config.Subscriptions {
		providerNames[subscription.Name] = struct{}{}
		if subscription.Enabled {
			enabledProviders[subscription.Name] = struct{}{}
		}
	}

	views := make([]TransitRouteView, 0, len(s.config.TransitRoutes))
	for _, route := range s.config.TransitRoutes {
		views = append(views, s.buildTransitRouteView(route, groupsByName, providerNames, enabledProviders))
	}
	return views
}

func (s *Service) getTransitRouteViewByName(name string) *TransitRouteView {
	for _, route := range s.GetTransitRoutes() {
		if route.Name == name {
			routeCopy := route
			return &routeCopy
		}
	}
	return nil
}

func (s *Service) GetListeners() []ListenerView {
	groupMap := map[string]GroupView{}
	for _, group := range s.GetGroups() {
		groupMap[group.Name] = group
	}
	transitRouteMap := map[string]TransitRouteView{}
	for _, route := range s.GetTransitRoutes() {
		transitRouteMap[route.Name] = route
	}
	groupNames := map[string]struct{}{}
	for _, group := range s.config.EgressGroups {
		groupNames[group.Name] = struct{}{}
	}
	transitRouteNames := map[string]struct{}{}
	for _, route := range s.config.TransitRoutes {
		transitRouteNames[route.Name] = struct{}{}
	}

	listeners := make([]ListenerView, 0, len(s.config.Listeners))
	for _, listener := range s.config.Listeners {
		routeMode := getListenerRouteMode(listener)
		group, groupOK := groupMap[listener.EgressGroup]
		_, groupExists := groupNames[listener.EgressGroup]
		transitRoute, transitOK := transitRouteMap[listener.TransitRoute]
		_, transitExists := transitRouteNames[listener.TransitRoute]

		status := "configured"
		groupMissing := false
		providerMissing := false
		providerDisabled := false
		landingMissing := false
		landingDisabled := false
		transitMissing := false
		transitDisabled := false
		transitProxyMissing := false
		currentProxy := ""
		routeSummary := ""
		egressGroup := listener.EgressGroup
		targetName := listener.EgressGroup

		if routeMode == "transit" {
			transitMissing = !transitExists
			transitDisabled = transitOK && transitRoute.Status == "disabled"
			transitProxyMissing = transitRoute.TransitProxyMissing
			providerMissing = transitRoute.EgressProviderMissing
			providerDisabled = transitRoute.EgressProviderDisabled
			landingMissing = transitRoute.LandingMissing
			landingDisabled = transitRoute.LandingDisabled
			currentProxy = transitRoute.CurrentProxy
			routeSummary = transitRoute.RouteSummary
			egressGroup = transitRoute.EgressGroup
			targetName = firstNonEmpty(transitRoute.Name, listener.TransitRoute)

			switch {
			case transitMissing:
				status = "orphaned"
			case transitDisabled:
				status = "disabled"
			case transitOK:
				status = transitRoute.Status
			default:
				status = "degraded"
			}
		} else {
			groupMissing = !groupExists
			providerMissing = group.ProviderMissing
			providerDisabled = group.ProviderDisabled
			landingMissing = group.LandingMissing
			landingDisabled = group.LandingDisabled
			if groupOK {
				currentProxy = group.Current
				routeSummary = group.RouteSummary
			}

			if groupMissing {
				status = "orphaned"
			} else if providerMissing || providerDisabled || landingMissing || landingDisabled {
				status = "degraded"
			}
		}

		listeners = append(listeners, ListenerView{
			Name:                listener.Name,
			Type:                firstNonEmpty(listener.Type, "socks"),
			Listen:              firstNonEmpty(listener.Listen, "0.0.0.0"),
			Port:                listener.Port,
			UDP:                 listener.UDP,
			Enabled:             listener.Enabled,
			Users:               listener.Users,
			UserCount:           len(listener.Users),
			RouteMode:           routeMode,
			EgressGroup:         egressGroup,
			TransitRoute:        listener.TransitRoute,
			TargetName:          targetName,
			CurrentProxy:        currentProxy,
			RouteSummary:        firstNonEmpty(routeSummary, targetName),
			Status:              status,
			GroupMissing:        groupMissing,
			ProviderMissing:     providerMissing,
			ProviderDisabled:    providerDisabled,
			LandingMissing:      landingMissing,
			LandingDisabled:     landingDisabled,
			TransitMissing:      transitMissing,
			TransitDisabled:     transitDisabled,
			TransitProxyMissing: transitProxyMissing,
		})
	}
	return listeners
}

func (s *Service) buildLandingProxyView(landing LandingProxy) LandingProxyView {
	inUseBy := []string{}
	for _, group := range s.config.EgressGroups {
		if group.LandingProxy == landing.Name {
			inUseBy = append(inUseBy, group.Name)
		}
	}
	return LandingProxyView{
		Name:           landing.Name,
		Type:           landing.Type,
		Server:         landing.Server,
		Port:           landing.Port,
		Username:       landing.Username,
		Password:       landing.Password,
		TLS:            landing.TLS,
		SNI:            landing.SNI,
		SkipCertVerify: landing.SkipCertVerify,
		Enabled:        landing.Enabled,
		InUseBy:        inUseBy,
		RouteCount:     len(inUseBy),
	}
}

func (s *Service) GetLandingProxies() []LandingProxyView {
	views := make([]LandingProxyView, 0, len(s.config.LandingProxies))
	for _, landing := range s.config.LandingProxies {
		views = append(views, s.buildLandingProxyView(landing))
	}
	return views
}
