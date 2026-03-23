package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "metalogs",
	Short: "Local dev logging system backed by SQLite",
}

func init() {
	defaultDB, _ := defaultDBPath()
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(wipeCmd)
	rootCmd.AddCommand(sitesCmd)
	rootCmd.AddCommand(collectionsCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.metalogs/metalogs.db", nil
}
