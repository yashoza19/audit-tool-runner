package main

import (
	"context"
	"encoding/json"
	"github.com/gobuffalo/envy"
	"github.com/google/uuid"
	"github.com/itchyny/gojq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func auditCmd() *cobra.Command {
	audit := &cobra.Command{
		Use:     "audit",
		Short:   "",
		Long:    "",
		PreRunE: validation,
		RunE:    run,
	}

	audit.Flags().StringVar(&flags.IndexImage, "index-image", "",
		"Certification index available to pull from public or private registry. Pulling from a private "+
			"registry requires setting the --registry-pull-secret flag")
	audit.Flags().StringVar(&flags.RegistryPullSecret, "registry-pull-secret", "",
		"Name of Kubernetes Secret to use for pulling registry images")

	return audit
}

var flags RunnerFlags

func validation(cmd *cobra.Command, args []string) error {
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	runTimeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	log.Infof("Starting operator-audit-%s run.\n", runTimeStamp)

	// Get list of bundles from an index
	cmdCreateBundleList := exec.Command("audit-tool-orchestrator", "index", "bundles",
		"--index-image", "registry.redhat.io/redhat/certified-operator-index:v4.9",
		"--container-engine", envy.Get("CONTAINER_ENGINE", "podman"))

	err := cmdCreateBundleList.Run()
	if err != nil {
		log.Fatalf("Unable to create bundlelist.json; audit-tool-orchestrator failed: %v\n", err)
	}

	// Create bucket to store test logs
	endpoint := envy.Get("MINIO_ENDPOINT", "")
	accessKeyID := envy.Get("MINIO_ACCESS_KEY", "")
	secretAccessKey := envy.Get("MINIO_SECRET_ACCESS_KEY", "")
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalf("Unable to initialize a minio client: %v\n", err)
	}

	bucket := "operator-audit-" + runTimeStamp
	log.Infof("Creating bucket %s to store logs\n", bucket)

	if err := minioClient.MakeBucket(context.TODO(), bucket, minio.MakeBucketOptions{}); err != nil {
		log.Fatalf("Unable to create minio bucket to store logs: %v\n", err)
	}

	log.Infof("Created bucket %s for audit.\n", bucket)

	// Create cluster pool for run
	log.Infof("Creating ClusterPool resource for audit.")

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
	if err = json.Unmarshal(bundledata, &bundlelist); err != nil {
		log.Fatalf("Unable to unmarshal bundledata: %v\n", err)
	}
	bundlecount, ok := query.Run(bundlelist).Next()
	if !ok {
		log.Fatalf("Unable to return a bundle count")
	}

	var poolSize, poolRunning string

	switch bundlecount {
	case bundlecount.(int) < 10:
		poolSize = bundlecount.(string)
		poolRunning = bundlecount.(string)
	case bundlecount.(int) > 10 && bundlecount.(int) < 30:
		poolSize = bundlecount.(string)
		poolRunning = string(bundlecount.(int) / 2)
	default:
		poolSize = "9"
		poolRunning = "3"
	}

	log.Info(poolSize, poolRunning)

	// Create cluster pool
	cpName := "operator-audit-" + runTimeStamp

	cmdCreatePool := exec.Command("audit-tool-orchestrator", "orchestrate", "pool",
		"--install-config", "sno-install-config",
		"--credentials", "hive-aws-creds",
		"--name", cpName,
		"--openshift", "4.9",
		"--platform", "aws",
		"--region", "us-east-1",
		"--size", poolSize,
		"--running", poolRunning)

	err = cmdCreatePool.Run()
	if err != nil {
		log.Fatalf("Unable to create the cluster pool: %v\n", err)
	}

	log.Info("ClusterPool created.")

	// Batch run
	log.Info("Setup complete starting batch runs.")

	// Parse list of bundles returning list for audit
	auditQuery, err := gojq.Parse(".Bundles")
	if err != nil {
		log.Errorf("Unable to parse bundle list json: %v\n", err)
	}

	auditlist, ok := auditQuery.Run(bundlelist).Next()
	if !ok {
		log.Fatalf("Unable to return a bundle count")
	}

	// Loop through chunks of 10 running audit tool waiting for result (Completed or Failed)
	batch := 3
	ch := make(chan string, 3)
	// TODO: set this back to actual bundle count via length
	runCount := 6

	for i := 0; i < runCount; i += batch {
		j := i + batch
		if j > runCount {
			j = runCount
		}

		for _, optoaudit := range auditlist.([]interface{})[i:j] {
			uniqueId := uuid.New()
			auditName := "operator-audit-" + strings.Split(uniqueId.String(), "-")[0]
			claimflags := ClaimFlags{
				Name:       auditName,
				Namespace:  "hive",
				PoolName:   cpName,
				BundleName: optoaudit.(map[string]interface{})["name"].(string),
				Delete:     false,
			}
			jobflags := JobFlags{
				Name:        auditName,
				BundleImage: optoaudit.(map[string]interface{})["bundleImage"].(string),
				BundleName:  optoaudit.(map[string]interface{})["name"].(string),
				BucketName:  bucket,
				ClaimName:   auditName,
				Kubeconfig:  auditName,
			}

			go RunOperatorAudit(ch, claimflags, jobflags)
		}

		for _, opaudited := range auditlist.([]interface{})[i:j] {
			runStatus := <-ch
			log.Infof("Operator: %s, Status: %s", opaudited.(map[string]interface{})["name"].(string), runStatus)
		}
	}

	log.Infof("Deleting cluster pool %s.\n", cpName)
	hvclient := GetHiveClient()

	payload := []PatchValue{{
		Op:    "replace",
		Path:  "/spec/size",
		Value: uint32(0),
	}, {
		Op:    "replace",
		Path:  "/spec/runningCount",
		Value: uint32(0),
	}}
	payloadBytes, _ := json.Marshal(payload)
	_, err = hvclient.HiveV1().ClusterPools("hive").Patch(context.TODO(), cpName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		log.Errorf("Unable to update cluster pool %s for deletion: %v\n", cpName, err)
	}

	err = hvclient.HiveV1().ClusterPools("hive").Delete(context.TODO(), cpName, metav1.DeleteOptions{})
	if err != nil {
		log.Errorf("Unable to delete the cluster pool %s: %v\n", cpName, err)
	}

	return nil
}
