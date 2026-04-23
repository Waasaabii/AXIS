package axis

func (s *Service) cloneConfig() Config {
	return mustJSONClone(*s.config)
}

func (s *Service) hasSubscription(name string) bool {
	for _, subscription := range s.config.Subscriptions {
		if subscription.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) findSubscription(name string) *Subscription {
	for index := range s.config.Subscriptions {
		if s.config.Subscriptions[index].Name == name {
			return &s.config.Subscriptions[index]
		}
	}
	return nil
}

func (s *Service) hasLandingProxy(name string) bool {
	for _, landing := range s.config.LandingProxies {
		if landing.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) hasTransitRoute(name string) bool {
	for _, route := range s.config.TransitRoutes {
		if route.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) hasGroup(name string) bool {
	for _, group := range s.config.EgressGroups {
		if group.Name == name {
			return true
		}
	}
	return false
}
