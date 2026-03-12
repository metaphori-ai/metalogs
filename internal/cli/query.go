package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
	"github.com/spf13/cobra"
)

var (
	querySite       string
	queryLayer      string
	queryCollection string
	queryLevel      string
	querySince      string
	queryContains   string
	queryLimit      int
	queryOffset     int
	queryJSON       bool
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logs",
	RunE:  runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&querySite, "site", "", "filter by site")
	queryCmd.Flags().StringVar(&queryLayer, "layer", "", "filter by layer")
	queryCmd.Flags().StringVar(&queryCollection, "collection", "", "filter by collection name")
	queryCmd.Flags().StringVar(&queryLevel, "level", "", "filter by level (comma-separated)")
	queryCmd.Flags().StringVar(&querySince, "since", "", "logs since duration (e.g. 1h, 7d) or RFC3339")
	queryCmd.Flags().StringVar(&queryContains, "contains", "", "filter by message substring")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 50, "max results")
	queryCmd.Flags().IntVar(&queryOffset, "offset", 0, "result offset")
	queryCmd.Flags().BoolVar(&queryJSON, "json", false, "output as JSON")
}

func runQuery(cmd *cobra.Command, args []string) error {
	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	opts := metalogs.QueryOpts{
		Site:       querySite,
		Layer:      queryLayer,
		Collection: queryCollection,
		Contains:   queryContains,
		Limit:      queryLimit,
		Offset:     queryOffset,
	}

	if queryLevel != "" {
		for _, l := range strings.Split(queryLevel, ",") {
			opts.Levels = append(opts.Levels, metalogs.LogLevel(strings.TrimSpace(l)))
		}
	}

	if querySince != "" {
		if d, err := parseDuration(querySince); err == nil {
			t := time.Now().UTC().Add(-d)
			opts.Since = &t
		} else if t, err := time.Parse(time.RFC3339, querySince); err == nil {
			opts.Since = &t
		} else {
			return fmt.Errorf("invalid --since value: %s", querySince)
		}
	}

	results, err := store.Query(opts)
	if err != nil {
		return err
	}

	if queryJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("no logs found")
		return nil
	}

	for _, e := range results {
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		src := ""
		if e.Source != nil {
			src = " [" + *e.Source + "]"
		}
		fmt.Printf("%s  %-5s  %-16s %-10s  %s%s\n", ts, strings.ToUpper(string(e.Level)), e.Site, e.Layer, e.Message, src)
		if e.Details != nil {
			fmt.Printf("         details: %s\n", *e.Details)
		}
	}

	return nil
}
