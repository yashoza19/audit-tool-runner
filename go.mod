module audit-tool-runner

go 1.16

require (
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
)

require (
	github.com/gobuffalo/envy v1.10.1
	github.com/google/uuid v1.1.2
	github.com/itchyny/gojq v0.12.7
	github.com/minio/minio-go/v7 v7.0.23
	github.com/openshift/hive v1.1.16
	github.com/openshift/hive/apis v0.0.0
	k8s.io/api v0.23.5 // indirect
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v12.0.0+incompatible
)

replace github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20220309220625-f517f1ce231e

replace k8s.io/client-go => k8s.io/client-go v0.23.4

// from installer
replace (
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3 // indirect
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // indirect
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d // indirect
	github.com/terraform-providers/terraform-provider-aws => github.com/openshift/terraform-provider-aws v1.60.1-0.20200630224953-76d1fb4e5699 // indirect
	github.com/terraform-providers/terraform-provider-azurerm => github.com/openshift/terraform-provider-azurerm v1.40.1-0.20200707062554-97ea089cc12a // indirect
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0 // indirect
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0 // indirect
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201022175424-d30c7a274820 // indirect
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201016155852-4090a6970205 // indirect
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20201116051540-155384b859c5 // indirect
)
