package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var wipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Delete the entire database and start fresh",
	RunE:  runWipe,
}

func runWipe(cmd *cobra.Command, args []string) error {
	files := []string{
		dbPath,
		dbPath + "-wal",
		dbPath + "-shm",
	}

	removed := 0
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("remove %s: %w", f, err)
		}
		removed++
	}

	if removed == 0 {
		fmt.Println("nothing to wipe")
	} else {
		fmt.Printf("wiped %s (%d files removed)\n", dbPath, removed)
	}
	return nil
}
