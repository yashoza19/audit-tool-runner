package main

import "github.com/openshift/hive/apis/hive/v1/azure"

type RunnerFlags struct {
	IndexImage         string `json:"indexImage"`
	RegistryPullSecret string `json:"registryPullSecret"`
}

type PoolFlags struct {
	Name                             string                 `json:"name"`
	Namespace                        string                 `json:"namespace"`
	BaseDomain                       string                 `json:"baseDomain"`
	OpenShift                        string                 `json:"openshift"`
	InstallConfig                    string                 `json:"installConfig"`
	ImagePullSecret                  string                 `json:"image-pull-secret"`
	Platform                         string                 `json:"platform"`
	Credentials                      string                 `json:"credentials"`
	Region                           string                 `json:"region"`
	Running                          int32                  `json:"running"`
	Size                             int32                  `json:"size"`
	AzureBaseDomainResourceGroupName string                 `json:"azurebasedomainresourcegroupname"`
	AzureCloudName                   azure.CloudEnvironment `json:"azurecloudname"`
	IBMAccountID                     string                 `json:"ibmaccountid"`
	IBMCISInstanceCRN                string                 `json:"ibmcisinstancecrn"`
}

type ClaimFlags struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	PoolName   string `json:"poolName"`
	BundleName string `json:"bundleName"`
	Delete     bool   `json:"delete"`
}

type JobFlags struct {
	Name        string `json:"name"`
	BundleImage string `json:"bundleImage"`
	BundleName  string `json:"bundleName"`
	BucketName  string `json:"bucketName"`
	ClaimName   string `json:"claimName"`
	Kubeconfig  string `json:"kubeconfig"`
}
