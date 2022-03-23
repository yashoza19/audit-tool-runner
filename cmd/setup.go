package main

import (
	"context"
	"encoding/json"
	"github.com/gobuffalo/envy"
	"github.com/itchyny/gojq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	hivev1api "github.com/openshift/hive/apis/hive/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"time"
)

func setupCmd() *cobra.Command {
	setup := &cobra.Command{
		Use:     "setup",
		Short:   "",
		Long:    "",
		PreRunE: validation,
		RunE:    run,
	}

	return setup
}

func validation(cmd *cobra.Command, args []string) error {
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	runTimeStamp := string(time.Now().Unix())
	log.Infof("Starting audit run %s\n", runTimeStamp)

	// Get list of bundles from an index
	cmdCreateBundleList := exec.Command("audit-tool-orchestrator", "index", "bundles",
		"--index-image", envy.Get("INDEX_IMAGE", "registry.redhat.io/redhat/certified-operator-index:v4.9"),
		"--container-engine", envy.Get("CONTAINER_ENGINE", "podman"))

	err := cmdCreateBundleList.Run()
	if err != nil {
		log.Fatalf("Unable to create bundlelist.json; audit-tool-orchestrator failed: %v\n", err)
	}

	// Create bucket to store test logs
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
	secretAccessKey := os.Getenv("MINIO_SECRET_ACCESS_KEY")
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalf("Unable to initialize a minio client: %v\n", err)
	}

	bucket := "operator-audit-" + runTimeStamp

	if err := minioClient.MakeBucket(context.TODO(), bucket, minio.MakeBucketOptions{}); err != nil {
		log.Fatalf("Unable to create minio bucket to store logs: %v\n", err)
	}

	log.Infof("Created bucket %s for audit.\n", bucket)

	// Create cluster pool for run
	fBundleList, err := os.Open("bundlelist.json")
	if err != nil {
		log.Fatalf("Unable to open bundlelist.json: %v\n", err)
	}
	defer func(fBundleList *os.File) {
		err = fBundleList.Close()
		if err != nil {
			log.Fatalf("bundlelist.json not closed check for memory leak!: %v\n", err)
		}
	}(fBundleList)
	bundledata, err := ioutil.ReadAll(fBundleList)
	if err != nil {
		log.Fatalf("Unable to read bundlelist.json: %v\n", err)
	}

	query, err := gojq.Parse(".Bundles | length")
	if err != nil {
		log.Errorf("Unable to parse bundle list json: %v\n", err)
	}

	bundlelist := make(map[string]interface{})
	if err := json.Unmarshal(bundledata, &bundlelist); err != nil {
		log.Fatalf("Unable to unmarshal bundledata: %v\n", err)
	}
	bundlecount, ok := query.Run(bundlelist).Next()
	if !ok {
		log.Fatalf("Unable to return a bundle count")
	}

	var poolSize, poolRunning string

	if bundlecount.(int) < 30 {
		poolSize = string(bundlecount.(int))
		poolRunning = string(bundlecount.(int) / 2)
	} else {
		poolSize = "30"
		poolRunning = "10"
	}

	// TODO: use these in cmdCreatePool
	log.Infof("PoolSize: %s, PoolRunning: %s\n", poolSize, poolRunning)

	// Create cluster pool
	cpName := "audit-tool-runner-" + runTimeStamp
	cmdCreatePool := exec.Command("audit-tool-orchestrator", "orchestrate", "pool",
		"--install-config", "sno-install-config",
		"--credentials", "hive-aws-creds",
		"--name", cpName,
		"--openshift", "4.9",
		"--platform", "aws",
		"--region", "us-east-1",
		"--running", "1", //poolRunning,
		"--size", "2") //poolSize

	err = cmdCreatePool.Run()
	if err != nil {
		log.Fatalf("Unable to create the cluster pool: %v\n", err)
	}

	cp := hivev1api.ClusterPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cpName,
			Namespace: "hive",
		},
	}

	hvclient := GetHiveClient()

	// ClusterPool resource submitted check it is available
	cpStatus, err := WaitForSuccessfulClusterPool(hvclient, &cp)
	if err != nil {
		log.Fatalf("ClusterPool returned an error: %v\n", err)
	}

	if cpStatus != "Pool Ready" {
		log.Fatalf("ClusterPool created but not ready after waiting for successful status.")
	}

	// Batch run

	return nil
}
