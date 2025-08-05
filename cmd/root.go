package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "messaging",
	Short: "Multi-Tenant Messaging System",
	Long:  `Backend system with dynamic consumer management and partitioned storage per tenant.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
