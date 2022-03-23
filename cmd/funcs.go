package main

import (
	"context"
	"fmt"
	hivev1api "github.com/openshift/hive/apis/hive/v1"
	hivev1client "github.com/openshift/hive/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

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

func WaitForSuccessfulClusterPool(hvclient *hivev1client.Clientset, pool *hivev1api.ClusterPool) (string, error) {
	ctx := context.Background()
	selector := fields.SelectorFromSet(map[string]string{"metadata.name": pool.Name})
	var wi watch.Interface

	err := wait.ExponentialBackoff(
		wait.Backoff{Steps: 10, Duration: 10 * time.Second, Factor: 2},
		func() (bool, error) {
			var err error
			cci := hvclient.HiveV1().ClusterPools(pool.Namespace)

			wi, err = cci.Watch(ctx, metav1.ListOptions{FieldSelector: selector.String()})
			if err != nil {
				log.Error(err)
				return false, nil
			}

			return true, nil
		},
	)

	if err != nil {
		log.WithError(err).Fatal("failed to create watch for ClusterPool")
		return "Pool Not Ready", err
	}

	for event := range wi.ResultChan() {
		clusterPool, ok := event.Object.(*hivev1api.ClusterPool)
		if !ok {
			log.WithField("object-type", fmt.Sprintf("%T", event.Object)).Fatal("received an unexpected object from Watch")
		}

		log.Infof("ClusterPool event received: %v\n", clusterPool.Status)

		poolStatusReady := clusterPool.Status.Ready
		poolStatusSize := clusterPool.Status.Size
		poolSpecRunning := clusterPool.Spec.RunningCount
		poolSpecSize := clusterPool.Spec.Size

		if poolStatusReady == poolSpecRunning && poolStatusSize == poolSpecSize {
			watchedPool, err := hvclient.HiveV1().ClusterPools(pool.Namespace).Get(ctx, pool.Name, metav1.GetOptions{})
			if err != nil {
				log.Errorf("Unable to get the ClusterPool under watch: %v\n", err)
				return "Pool Not Ready", err
			}
			log.Info(watchedPool)
			return "Pool Ready", nil
		}
	}

	return "Pool Not Ready", err
}
