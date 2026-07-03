package worker

import (
	"errors"

	"github.com/whs/hordebridge/aihorde"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/semconv/v1.41.0/messagingconv"
)

const otelPackageName = "whs.in.th/hordebridge"

var messagingSystemHorde = messagingconv.SystemAttr("horde")
var keyKoboldState = attribute.Key("state")
var keyInputLength = attribute.Key("length")
var keyClassifierResult = attribute.Key("result")
var keyClassifierBlocked = attribute.Key("blocked")

// gen ai event semconv spec

var keyGenAiRequestChoiceCount = attribute.Key("gen_ai.request.choice.count")

func (w *Worker) initOtel() error {
	var outErr error
	var err error
	meter := otel.Meter(otelPackageName)

	w.metricClientConsumedMessages, err = messagingconv.NewClientConsumedMessages(meter)
	outErr = errors.Join(outErr, err)

	w.metricJobProcessingDuration, err = messagingconv.NewProcessDuration(meter)
	outErr = errors.Join(outErr, err)

	w.metricKudos, err = meter.Float64Counter(
		"kudos",
		metric.WithDescription("Kudos received"),
		metric.WithUnit("{kudo}"),
	)
	outErr = errors.Join(outErr, err)

	w.metricInputLength, err = meter.Int64Counter(
		"input.size",
		metric.WithDescription("Input characters processed"),
		metric.WithUnit("{character}"),
	)
	outErr = errors.Join(outErr, err)

	w.metricOutputLength, err = meter.Int64Counter(
		"output.size",
		metric.WithDescription("Output tokens processed"),
		metric.WithUnit("{character}"),
	)
	outErr = errors.Join(outErr, err)

	w.metricClassifierResult, err = meter.Int64Counter(
		"classifier.result",
		metric.WithDescription("Classifier results"),
		metric.WithUnit("{job}"),
	)
	outErr = errors.Join(outErr, err)

	w.metricClassifierDuration, err = meter.Float64Histogram(
		"classifier.duration",
		metric.WithDescription("Time used by classifier"),
		metric.WithUnit("s"),
	)
	outErr = errors.Join(outErr, err)

	return outErr
}

func inputToEventAttr(input *aihorde.ModelPayloadKobold) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0)

	if val, ok := input.Frmtadsnsp.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.frmtadsnsp", val))
	}
	if val, ok := input.Frmtrmblln.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.frmtrmblln", val))
	}
	if val, ok := input.Frmtrmspch.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.frmtrmspch", val))
	}
	if val, ok := input.Frmttriminc.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.frmttriminc", val))
	}
	if val, ok := input.RepPen.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.rep_pen", val))
	}
	if val, ok := input.RepPenRange.Get(); ok {
		out = append(out, attribute.Int("gen_ai.request.rep_pen_range", val))
	}
	if val, ok := input.RepPenSlope.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.rep_pen_slope", val))
	}
	if val, ok := input.Singleline.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.singleline", val))
	}
	if val, ok := input.Temperature.Get(); ok {
		out = append(out, semconv.GenAIRequestTemperature(val))
	}
	if val, ok := input.Tfs.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.tfs", val))
	}
	if val, ok := input.TopA.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.top_a", val))
	}
	if val, ok := input.TopK.Get(); ok {
		out = append(out, semconv.GenAIRequestTopK(float64(val)))
	}
	if val, ok := input.TopP.Get(); ok {
		out = append(out, semconv.GenAIRequestTopP(val))
	}
	if val, ok := input.Typical.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.typical", val))
	}
	out = append(out, attribute.IntSlice("gen_ai.request.sampler_order", input.SamplerOrder))
	if val, ok := input.UseDefaultBadwordsids.Get(); ok {
		out = append(out, attribute.Bool("gen_ai.request.use_default_badwordsids", val))
	}
	out = append(out, semconv.GenAIRequestStopSequences(input.StopSequence...))
	if val, ok := input.MinP.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.min_p", val))
	}
	if val, ok := input.SmoothingFactor.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.smoothing_factor", val))
	}
	if val, ok := input.DynatempRange.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.dynatemp_range", val))
	}
	if val, ok := input.DynatempExponent.Get(); ok {
		out = append(out, attribute.Float64("gen_ai.request.dynatemp_exponent", val))
	}
	if val, ok := input.N.Get(); ok {
		out = append(out, semconv.GenAIRequestChoiceCount(val))
	}
	if val, ok := input.MaxContextLength.Get(); ok {
		out = append(out, semconv.GenAIRequestChoiceCount(val))
	}
	if val, ok := input.MaxLength.Get(); ok {
		out = append(out, semconv.GenAIRequestMaxTokens(val))
	}

	return out
}
