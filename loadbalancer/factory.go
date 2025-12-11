package loadbalancer

func New(configType string, servers []string, weights []int) LoadBalancer {
	if weights == nil {
		f, ok := Registry[configType]
		if ok {
			return f(servers, weights)
		}
		return nil
	}

	f, ok := Registry[configType]
	if ok {
		return f(servers, weights)
	}
	return nil
}
