// Package telemetry exports a server's slog lines over OTLP, beside the handler they already go to.
package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"github.com/stupside/castor/internal/castid"
)

// Config is where a server sends its logs; empty, they stay on its own output.
type Config struct {
	Endpoint string `yaml:"endpoint" validate:"omitempty,http_url"`
}

// Setup adds an OTLP/HTTP exporter to the default logger as service, and returns what flushes it.
func Setup(ctx context.Context, c Config, service string) (func(context.Context) error, error) {
	if c.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(c.Endpoint))
	if err != nil {
		return nil, err
	}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(resource.NewSchemaless(semconv.ServiceName(service))),
	)
	slog.SetDefault(slog.New(slog.NewMultiHandler(slog.Default().Handler(), castid.Tagged(otelslog.NewHandler(service, otelslog.WithLoggerProvider(provider))))))
	return provider.Shutdown, nil
}
