package main

import "github.com/spf13/cobra"

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
	// Get bundles: audit-tool-orchestrator index bundles --index-image registry.redhat.io/redhat/certified-operator-index:v4.9 --container-engine podman
	// Create bucket: https://dl.min.io/client/mc/release/linux-amd64/mc
	return nil
}
