package cli

import (
	"fmt"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
	"github.com/spf13/cobra"
)

var siteFilter string

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "List sites and layers",
	RunE:  runSites,
}

func init() {
	sitesCmd.Flags().StringVar(&siteFilter, "site", "", "show layers for a specific site")
}

func runSites(cmd *cobra.Command, args []string) error {
	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if siteFilter != "" {
		layers, err := store.ListLayers(siteFilter)
		if err != nil {
			return err
		}
		if len(layers) == 0 {
			fmt.Printf("no layers found for site %q\n", siteFilter)
			return nil
		}
		for _, l := range layers {
			fmt.Printf("%s / %s\n", siteFilter, l)
		}
		return nil
	}

	pairs, err := store.ListSiteLayers()
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		fmt.Println("no sites found")
		return nil
	}

	for _, p := range pairs {
		fmt.Printf("%s / %s\n", p.Site, p.Layer)
	}
	return nil
}
