package cli

import (
	"fmt"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
	"github.com/spf13/cobra"
)

var cleanupOlderThan string

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete old logs",
	RunE:  runCleanup,
}

func init() {
	cleanupCmd.Flags().StringVar(&cleanupOlderThan, "older-than", "7d", "delete logs older than (e.g. 7d, 24h)")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ttl, err := parseDuration(cleanupOlderThan)
	if err != nil {
		return fmt.Errorf("invalid --older-than value: %w", err)
	}

	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	deleted, err := store.Cleanup(ttl)
	if err != nil {
		return err
	}

	fmt.Printf("deleted %d logs older than %s\n", deleted, cleanupOlderThan)
	return nil
}
