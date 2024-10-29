package config

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func InitOpenTelemetrySDK(ctx context.Context, instanceID, APIToken string) (shutdown func(context.Context) error, err error) {
	headers := make(map[string]string)
	credentials := base64.StdEncoding.EncodeToString([]byte(instanceID + ":" + APIToken))
	headers["Authorization"] = fmt.Sprintf("Basic %s", credentials)

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	metricExporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithReader(metric.NewManualReader(metric.WithProducer(runtime.NewProducer()))),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	err = runtime.Start()
	if err != nil {
		handleErr(err)
		return nil, err
	}

	err = host.Start(host.WithMeterProvider(meterProvider))
	if err != nil {
		handleErr(err)
		return nil, err
	}

	traceExporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithSpanProcessor(bsp),
	)
	shutdownFuncs = append(shutdownFuncs, traceProvider.Shutdown)
	otel.SetTracerProvider(traceProvider)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return
}
