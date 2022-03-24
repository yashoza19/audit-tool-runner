package main

import (
	"context"
	hivev1client "github.com/openshift/hive/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
)

// TODO: function for logging error, writing "Error" to channel, and returning from RunOperatorAudit

func GetHiveClient() *hivev1client.Clientset {
	// create hive client
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Printf("Unable to build config from flags: %v\n", err)
	}

	hiveclient, err := hivev1client.NewForConfig(cfg)

	return hiveclient
}

func GetK8sClient() *kubernetes.Clientset {
	// create k8s client
	cfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("OPENSHIFT_KUBECONFIG"))
	if err != nil {
		log.Errorf("Unable to build config from flags: %v\n", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)

	return clientset
}

func K8sClientForAudit(kubeconfig []byte) *kubernetes.Clientset {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		log.Errorf("Unable to build config from kubeconfig: %v\n", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)

	return clientset
}

func GetJobStatus(job string) string {
	kubeconfig, err := os.ReadFile("/tmp/" + job)
	if err != nil {
		log.Errorf("Unable to get kubeconfig for job status check: %v\n", err)
		return "Error"
	}
	k8sClient := K8sClientForAudit(kubeconfig)
	auditJob, err := k8sClient.BatchV1().Jobs("default").Get(context.TODO(), job, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to get job for status check: %v\n", err)
		return "Error"
	}

	auditJobConditions := auditJob.Status.Conditions
	var jobCondition string
	for _, auditJobCondition := range auditJobConditions {
		if auditJobCondition.Type != "" && (auditJobCondition.Type == "Completed" || auditJobCondition.Type == "Failed") {
			jobCondition = string(auditJobCondition.Type)
		}
	}

	return jobCondition
}

func RunOperatorAudit(ch chan string, claimflags ClaimFlags, jobflags JobFlags) {
	ctx := context.Background()

	// Claim a cluster
	log.Infof("Claiming cluster for %s\n", claimflags.Name)
	cmdClaimCluster := exec.Command("audit-tool-orchestrator", "orchestrate", "claim",
		"--name", claimflags.Name,
		"--pool-name", claimflags.PoolName,
		"--bundle-name", claimflags.BundleName)

	err := cmdClaimCluster.Run()
	if err != nil {
		log.Errorf("Unable to claim cluster for audit %s: %v\n", claimflags.Name, err)
		ch <- "Error"
		return
	}

	// Get credentials for claimed cluster
	log.Infof("Cluster claimed for %s; getting credentials\n", claimflags.Name)

	hvclient := GetHiveClient()
	clusterClaim, err := hvclient.HiveV1().ClusterClaims("hive").Get(ctx, claimflags.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to get ClusterClaim: %v\n", err)
		ch <- "Error"
		return
	}
	cdNameNamespace := clusterClaim.Spec.Namespace
	clusterDeployment, err := hvclient.HiveV1().ClusterDeployments(cdNameNamespace).Get(ctx, cdNameNamespace, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to get ClusterDeployment: %s\n", cdNameNamespace)
		ch <- "Error"
		return
	}

	kubeconfigSecret := clusterDeployment.Spec.ClusterMetadata.AdminKubeconfigSecretRef

	k8sclient := GetK8sClient()
	kubeconfig, err := k8sclient.CoreV1().Secrets(cdNameNamespace).Get(ctx, kubeconfigSecret.Name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to get kubeconfig for cluster under test: %v\n", err)
		ch <- "Error"
		return
	}

	if err = os.WriteFile("/tmp/"+claimflags.Name, kubeconfig.Data["raw-kubeconfig"], 0644); err != nil {
		log.Errorf("Unable to create kubeconfig: %v\n", err)
		ch <- "Error"
		return
	}

	// Add logging configmap to claimed cluster
	log.Infof("Adding configmap for logging to cluster %s\n", cdNameNamespace)
	envvarConfigmap, err := k8sclient.CoreV1().ConfigMaps("hive").Get(ctx, "env-var", metav1.GetOptions{})
	if err != nil {
		log.Errorf("Unable to get logging Configmap: %v\n", err)
		ch <- "Error"
		return
	}

	logConfigmap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env-var",
			Namespace: "default",
		},
		Data: envvarConfigmap.Data,
	}

	auditClient := K8sClientForAudit(kubeconfig.Data["raw-kubeconfig"])
	_, err = auditClient.CoreV1().ConfigMaps("default").Create(ctx, &logConfigmap, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Unable to put logging Configmap in cluster under audit: %v\n", err)
		ch <- "Error"
		return
	}

	// Create Job in claimed cluster
	log.Infof("Creating job %s in cluster %s for audit %s\n", jobflags.Name, cdNameNamespace, claimflags.Name)
	cmdCreateJob := exec.Command("audit-tool-orchestrator", "orchestrate", "job",
		"--name", jobflags.Name,
		"--claim-name", claimflags.Name,
		"--bundle-image", jobflags.BundleImage,
		"--bundle-name", jobflags.BundleName,
		"--bucket-name", jobflags.BucketName,
		"--kubeconfig", "/tmp/"+claimflags.Name)

	err = cmdCreateJob.Run()
	if err != nil {
		log.Errorf("Unable to create a job for audit %s: %v\n", claimflags.Name, err)
		ch <- "Error"
		return
	}

	log.Infof("Job %s has finished; getting status.\n", claimflags.Name)
	ch <- GetJobStatus(jobflags.Name)
	log.Infof("Deleting cluster claim %s.\n", claimflags.Name)
	err = hvclient.HiveV1().ClusterClaims("hive").Delete(ctx, claimflags.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Unable to delete cluster claim %s: %v\n", claimflags.Name, err)
	}
	return
}
