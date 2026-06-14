package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/whs/hordebridge/worker"
)

var cli struct {
	Config   worker.Config `embed:""`
	LogLevel slog.Level    `default:"INFO" enum:"DEBUG,INFO,WARN,ERROR"`
}

func main() {
	kong.Parse(&cli, kong.DefaultEnvars(""))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     cli.LogLevel,
	})))
	instance, err := worker.NewWorker(cli.Config)
	if err != nil {
		panic(err)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-runCtx.Done()
		// Immediately restore ctrl-c behavior after the first one
		slog.Info("Abort requested. Send second abort to immediately exit.")
		stop()
	}()

	instance.Start(runCtx, context.Background())
}
