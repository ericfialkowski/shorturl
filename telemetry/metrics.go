package telemetry

import (
	"context"
	"log"
	"time"

	"github.com/ericfialkowski/shorturl/env"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	otlpEndpoint = env.StringOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	serviceName  = env.StringOrDefault("OTEL_SERVICE_NAME", "shorturl")
	enabled      = env.BoolOrDefault("OTEL_METRICS_ENABLED", true)
)

// Metrics holds all the OpenTelemetry metric instruments for the application.
type Metrics struct {
	Redirects       metric.Int64Counter
	UrlsCreated     metric.Int64Counter
	UrlsDeleted     metric.Int64Counter
	StatsRequests   metric.Int64Counter
	RequestDuration metric.Float64Histogram

	provider *sdkmetric.MeterProvider
}

// NewMetrics initializes the OpenTelemetry metrics provider and creates all metric instruments.
// Returns nil if metrics are disabled via OTEL_METRICS_ENABLED=false.
func NewMetrics(ctx context.Context) (*Metrics, error) {
	if !enabled {
		log.Println("OpenTelemetry metrics disabled")
		return nil, nil
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(trimScheme(otlpEndpoint)),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(15*time.Second),
			),
		),
	)

	otel.SetMeterProvider(provider)
	meter := provider.Meter(serviceName)

	redirects, err := meter.Int64Counter("shorturl.redirects",
		metric.WithDescription("Number of URL redirects performed"),
		metric.WithUnit("{redirect}"),
	)
	if err != nil {
		return nil, err
	}

	urlsCreated, err := meter.Int64Counter("shorturl.urls.created",
		metric.WithDescription("Number of new short URLs created"),
		metric.WithUnit("{url}"),
	)
	if err != nil {
		return nil, err
	}

	urlsDeleted, err := meter.Int64Counter("shorturl.urls.deleted",
		metric.WithDescription("Number of short URLs deleted"),
		metric.WithUnit("{url}"),
	)
	if err != nil {
		return nil, err
	}

	statsRequests, err := meter.Int64Counter("shorturl.stats.requests",
		metric.WithDescription("Number of stats endpoint requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram("shorturl.request.duration",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("OpenTelemetry metrics initialized (endpoint: %s, service: %s)", otlpEndpoint, serviceName)

	return &Metrics{
		Redirects:       redirects,
		UrlsCreated:     urlsCreated,
		UrlsDeleted:     urlsDeleted,
		StatsRequests:   statsRequests,
		RequestDuration: requestDuration,
		provider:        provider,
	}, nil
}

// Shutdown gracefully shuts down the metrics provider.
func (m *Metrics) Shutdown(ctx context.Context) error {
	if m == nil || m.provider == nil {
		return nil
	}
	log.Println("Shutting down OpenTelemetry metrics provider")
	return m.provider.Shutdown(ctx)
}

// trimScheme removes http:// or https:// prefix from the endpoint.
func trimScheme(endpoint string) string {
	if len(endpoint) > 8 && endpoint[:8] == "https://" {
		return endpoint[8:]
	}
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		return endpoint[7:]
	}
	return endpoint
}
