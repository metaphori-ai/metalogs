package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/metaphori-ai/metalogs/internal/server"
	"github.com/metaphori-ai/metalogs/pkg/metalogs"
	"github.com/spf13/cobra"
)

var (
	servePort int
	serveTTL  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server with background cleanup",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 9999, "port to listen on")
	serveCmd.Flags().StringVar(&serveTTL, "ttl", "7d", "cleanup TTL (e.g. 7d, 24h)")
}

func runServe(cmd *cobra.Command, args []string) error {
	ttl, err := parseDuration(serveTTL)
	if err != nil {
		return fmt.Errorf("invalid ttl %q: %w", serveTTL, err)
	}

	store, err := metalogs.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	srv := server.New(store, server.Config{
		Port:       servePort,
		CleanupTTL: ttl,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx)
	}()

	return srv.Start()
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
