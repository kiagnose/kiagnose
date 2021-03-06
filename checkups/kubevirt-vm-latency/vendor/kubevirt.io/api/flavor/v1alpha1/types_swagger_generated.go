// Code generated by swagger-doc. DO NOT EDIT.

package v1alpha1

func (VirtualMachineFlavor) SwaggerDoc() map[string]string {
	return map[string]string{
		"":     "VirtualMachineFlavor resource contains common VirtualMachine configuration\nthat can be used by multiple VirtualMachine resources.\n\n+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object\n+k8s:openapi-gen=true\n+genclient",
		"spec": "VirtualMachineFlavorSpec for the flavor",
	}
}

func (VirtualMachineFlavorList) SwaggerDoc() map[string]string {
	return map[string]string{
		"": "VirtualMachineFlavorList is a list of VirtualMachineFlavor resources.\n\n+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object\n+k8s:openapi-gen=true",
	}
}

func (VirtualMachineClusterFlavor) SwaggerDoc() map[string]string {
	return map[string]string{
		"":     "VirtualMachineClusterFlavor is a cluster scoped version of VirtualMachineFlavor resource.\n\n+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object\n+k8s:openapi-gen=true\n+genclient\n+genclient:nonNamespaced",
		"spec": "VirtualMachineFlavorSpec for the flavor",
	}
}

func (VirtualMachineClusterFlavorList) SwaggerDoc() map[string]string {
	return map[string]string{
		"": "VirtualMachineClusterFlavorList is a list of VirtualMachineClusterFlavor resources.\n\n+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object\n+k8s:openapi-gen=true",
	}
}

func (VirtualMachineFlavorSpec) SwaggerDoc() map[string]string {
	return map[string]string{
		"":    "VirtualMachineFlavorSpec\n\n+k8s:openapi-gen=true",
		"cpu": "+optional",
	}
}
