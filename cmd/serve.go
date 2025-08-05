package cmd

import (
	"github.com/fekalegi/multi-tenant-system/config"
	"github.com/fekalegi/multi-tenant-system/internal/app"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server and consumers",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadConfig()
		app.Start(cfg)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
