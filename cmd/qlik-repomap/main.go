package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/cli"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
