package loadbalancer

type LoadBalancer interface {
	NextServer() (string, error)
}
