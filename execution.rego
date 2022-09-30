package execution

deny[resource] {
	resource := input[_]
	resource.enabled
	resource.ingress.cidr == "0.0.0.0/0"
	resource.ingress.port <= 1024
}
