// Based on https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gin-gonic/gin/otelgin

package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerKey  = "otel-go-contrib-tracer"
	tracerName = "github.com/flf2ko/otel-example/middleware"
)

// Middleware returns middleware that will trace incoming requests.
// The service parameter should describe the name of the (virtual)
// server handling the request.
func Middleware(service string, startOpts ...oteltrace.SpanStartOption) gin.HandlerFunc {

	tracer := otel.GetTracerProvider().Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion("middleware_version:v0.0.1"),
	)

	propagators := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		c.Set(tracerKey, tracer)
		savedCtx := c.Request.Context()
		defer func() {
			c.Request = c.Request.WithContext(savedCtx)
		}()
		ctx := propagators.Extract(savedCtx, propagation.HeaderCarrier(c.Request.Header))

		// Setup default span start options
		if len(startOpts) == 0 {
			startOpts = []oteltrace.SpanStartOption{
				oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", c.Request)...),
				oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(c.Request)...),
				oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, c.FullPath(), c.Request)...),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			}
		}

		path := c.FullPath()
		if path == "" {
			path = fmt.Sprintf("HTTP %s route not found", c.Request.Method)
		}
		spanName := fmt.Sprintf("%s %s", c.Request.Method, path)

		ctx, span := tracer.Start(ctx, spanName, startOpts...)
		defer span.End()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		// serve the request to the next middleware
		c.Next()

		status := c.Writer.Status()
		attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(status)
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
		if len(c.Errors) > 0 {
			span.SetAttributes(attribute.String("gin.errors", c.Errors.String()))
		}
	}
}
