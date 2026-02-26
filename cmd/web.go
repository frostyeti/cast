package cmd

import (
	"log"

	"github.com/frostyeti/cast/internal/web"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the Cast web server and cron scheduler",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		addr, _ := cmd.Flags().GetString("addr")

		server := web.NewServer(addr, port)
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start web server: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.PersistentFlags().IntP("port", "p", 8080, "Port to listen on")
	webCmd.PersistentFlags().StringP("addr", "a", "127.0.0.1", "Address to listen on")
}
