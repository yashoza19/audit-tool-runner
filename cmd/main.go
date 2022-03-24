package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "audit-tool-runner",
		Short: "Run operator capabilities audit.",
		Long:  "",
	}

	rootCmd.AddCommand(auditCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
