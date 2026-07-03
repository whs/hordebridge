package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"
	ourotel "github.com/whs/hordebridge/otel"
	"github.com/whs/hordebridge/worker"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var cli struct {
	Config   worker.Config `embed:""`
	LogLevel slog.Level    `default:"INFO" enum:"DEBUG,INFO,WARN,ERROR"`
}

func main() {
	kong.Parse(&cli, kong.DefaultEnvars(""))

	// Setup otel. However, if otel is not configured we default to none instead of otlp
	if _, ok := os.LookupEnv("OTEL_TRACES_EXPORTER"); !ok {
		os.Setenv("OTEL_TRACES_EXPORTER", "none")
	}
	if _, ok := os.LookupEnv("OTEL_METRICS_EXPORTER"); !ok {
		os.Setenv("OTEL_METRICS_EXPORTER", "none")
	}
	if _, ok := os.LookupEnv("OTEL_LOGS_EXPORTER"); !ok {
		os.Setenv("OTEL_LOGS_EXPORTER", "fancyconsole")
	}

	autoexport.RegisterLogExporter("fancyconsole", ourotel.NewFancyConsoleLogExporter)
	logExporter, err := autoexport.NewLogExporter(context.Background())
	if err != nil {
		panic(fmt.Errorf("unable to setup metric exporter: %w", err))
	}
	logger := log.NewLoggerProvider(log.WithProcessor(log.NewSimpleProcessor(logExporter)))
	global.SetLoggerProvider(logger)
	slog.SetDefault(otelslog.NewLogger("whs.in.th/hordebridge"))

	spanExporter, err := autoexport.NewSpanExporter(context.Background())
	if err != nil {
		panic(fmt.Errorf("unable to setup span exporter: %w", err))
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
	)
	otel.SetTracerProvider(tracerProvider)

	metricReader, err := autoexport.NewMetricReader(context.Background())
	if err != nil {
		panic(fmt.Errorf("unable to setup metric exporter: %w", err))
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
	)
	otel.SetMeterProvider(meterProvider)

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

	<-runCtx.Done()

	stopCtx, stop2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop2()
	err = tracerProvider.Shutdown(stopCtx)
	if err != nil {
		slog.WarnContext(stopCtx, "Failed to stop tracer", err)
	}
	err = meterProvider.Shutdown(stopCtx)
	if err != nil {
		slog.WarnContext(stopCtx, "Failed to stop meter", err)
	}
	logger.Shutdown(stopCtx)
}
