module github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency

go 1.17

require github.com/stretchr/testify v1.7.1

// Kubevirt client-go dependencies
require (
	k8s.io/apimachinery v0.23.5
	kubevirt.io/client-go v0.53.0
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)
