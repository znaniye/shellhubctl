package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/znaniye/shellhub-tui/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cli.Execute(ctx); err != nil {
		return 1
	}

	return 0
}
