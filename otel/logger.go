package otel

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	log "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

var logLevelColor = map[log.Severity]color.Color{
	log.SeverityTrace: lipgloss.Black,
	log.SeverityDebug: lipgloss.BrightBlack,
	log.SeverityInfo:  lipgloss.Blue,
	log.SeverityWarn:  lipgloss.Yellow,
	log.SeverityError: lipgloss.Red,
	log.SeverityFatal: lipgloss.BrightRed,
}

type fancyConsoleExporter struct {
}

func NewFancyConsoleLogExporter(ctx context.Context) (sdklog.Exporter, error) {
	return fancyConsoleExporter{}, nil
}

func (f fancyConsoleExporter) Export(ctx context.Context, records []sdklog.Record) error {
	var err error
	for _, record := range records {
		err = errors.Join(f.export(record))
	}
	return err
}

func (f fancyConsoleExporter) export(record sdklog.Record) error {
	// [Time] [level] [module]  [msg]
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Width(len(time.TimeOnly)).MarginRight(1)
	levelColor, ok := logLevelColor[record.Severity()]
	if !ok {
		levelColor = lipgloss.BrightGreen
	}
	levelStyle := lipgloss.NewStyle().Bold(true).
		Width(len(log.SeverityDebug.String()) + 2).
		MarginRight(1).
		PaddingLeft(1).PaddingRight(1).
		Background(levelColor).Foreground(lipgloss.Lighten(lipgloss.Complementary(levelColor), 0.4)).
		AlignHorizontal(lipgloss.Center)
	moduleStyle := lipgloss.NewStyle().Foreground(lipgloss.Yellow).MarginRight(1)
	kvStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)

	var module string
	var kvList []string
	record.WalkAttributes(func(kv log.KeyValue) bool {
		if kv.Key == "module" {
			module = kv.Value.String()
		} else {
			kvList = append(kvList, fmt.Sprintf("%s=%s", kv.Key, kv.Value.String()))
		}
		return true
	})

	_, err := lipgloss.Println(lipgloss.JoinHorizontal(
		lipgloss.Top,
		timeStyle.Render(record.Timestamp().Format(time.TimeOnly)),
		levelStyle.Render(record.Severity().String()),
		moduleStyle.Render(module),
		record.Body().String(),
		kvStyle.MarginLeft(1).Render(strings.Join(kvList, " ")),
	))

	return err
}

func (f fancyConsoleExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (f fancyConsoleExporter) ForceFlush(ctx context.Context) error {
	return nil
}
