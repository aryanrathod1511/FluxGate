package loadbalancer

var Registry = make(map[string]func([]string) LoadBalancer)

func RegistrLoadBalancer(name string, constructor func([]string) LoadBalancer) {
	Registry[name] = constructor
}
