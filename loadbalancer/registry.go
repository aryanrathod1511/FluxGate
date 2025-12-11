package loadbalancer

var Registry = make(map[string]func([]string, []int) LoadBalancer)

func RegistrLoadBalancer(name string, constructor func([]string, []int) LoadBalancer) {
	Registry[name] = constructor
}
