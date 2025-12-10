package loadbalancer

func New(configType string, servers []string) LoadBalancer {
	f, ok := Registry[configType]
	if ok {
		return f(servers)
	}
	return nil
}
