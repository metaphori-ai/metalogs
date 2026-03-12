package cli

import (
	"fmt"
	"strings"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
	"github.com/spf13/cobra"
)

var collectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "Manage collections of site+layer pairs",
}

var collectionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	RunE:  runCollectionsList,
}

var collectionsCreateCmd = &cobra.Command{
	Use:   "create <name> <site:layer,site:layer,...>",
	Short: "Create a collection",
	Args:  cobra.ExactArgs(2),
	RunE:  runCollectionsCreate,
}

var collectionsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a collection",
	Args:  cobra.ExactArgs(1),
	RunE:  runCollectionsDelete,
}

func init() {
	collectionsCmd.AddCommand(collectionsListCmd)
	collectionsCmd.AddCommand(collectionsCreateCmd)
	collectionsCmd.AddCommand(collectionsDeleteCmd)
}

func runCollectionsList(cmd *cobra.Command, args []string) error {
	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	colls, err := store.ListCollections()
	if err != nil {
		return err
	}

	if len(colls) == 0 {
		fmt.Println("no collections found")
		return nil
	}

	for _, c := range colls {
		var members []string
		for _, m := range c.Members {
			members = append(members, m.Site+":"+m.Layer)
		}
		fmt.Printf("%-20s  %s\n", c.Name, strings.Join(members, ", "))
	}
	return nil
}

func runCollectionsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	parts := strings.Split(args[1], ",")

	var members []metalogs.SiteLayer
	for _, p := range parts {
		p = strings.TrimSpace(p)
		site, layer, ok := strings.Cut(p, ":")
		if !ok {
			return fmt.Errorf("invalid member %q, expected site:layer", p)
		}
		members = append(members, metalogs.SiteLayer{Site: site, Layer: layer})
	}

	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.CreateCollection(name, members); err != nil {
		return err
	}

	fmt.Printf("created collection %q with %d members\n", name, len(members))
	return nil
}

func runCollectionsDelete(cmd *cobra.Command, args []string) error {
	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.DeleteCollection(args[0]); err != nil {
		return err
	}

	fmt.Printf("deleted collection %q\n", args[0])
	return nil
}
