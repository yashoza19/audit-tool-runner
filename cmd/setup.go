package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strings"
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
	log.Infof("Starting audit run %v\n", time.Now().Format("Monday 01/02/2006 15:04:05"))

	// Get list of bundles from an index
	/*exec.Command("audit-tool-orchestrator", "index", "bundles",
	"--index-image", envy.Get("INDEX_IMAGE", "registry.redhat.io/redhat/certified-operator-index:v4.9"),
	"--container-engine", envy.Get("CONTAINER_ENGINE", "podman"))*/

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

	bucket := strings.ToLower("operator-audit-" + time.Now().Format("Monday-01-02-2006-15-04-05"))

	if err = minioClient.MakeBucket(context.TODO(), bucket, minio.MakeBucketOptions{}); err != nil {
		log.Fatalf("Unable to create minio bucket to store logs: %v\n", err)
	}

	log.Infof("Created bucket %s for audit.\n", bucket)

	// Create cluster pool for run
	/*exec.Command("audit-tool-orchestrator", "orchestrate", "pool",
	"--install-config", "sno-install-config",
	"--name", "",
	"--openshift", "4.9.23",
	"--platform", "aws",
	"--region", "us-east-1",
	"--running", "",
	"--size", "")*/

	// Batch run
	return nil
}
