package command

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	otelexporter "go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	customMiddleware "github.com/flf2ko/otel-example/middleware"
)

var (
	quit      = make(chan os.Signal, 5)
	systemCtx context.Context
	tracer    = otel.Tracer("gin-server")
)

func init() {
	systemCtx = context.Background()
}

func NewServerCmd() *cobra.Command {
	var serverConfigFile string
	var cmdAPI = &cobra.Command{
		Use:   "server",
		Short: "Start the server",
		Long:  `Run the http server otel-example`,
		Run: func(cmd *cobra.Command, args []string) {
			defer log.Println("server main thread exiting")
			log.Println("config path:", serverConfigFile)

			tp := initTracer()
			defer func() {
				if err := tp.Shutdown(context.Background()); err != nil {
					log.Printf("Error shutting down tracer provider: %v", err)
				}
			}()

			// Init HTTP server, to provide readiness information at the very beginning
			httpServer, err := InitGinServer(systemCtx)
			if err != nil {
				panic("api server init error:" + err.Error())
			}

			defer func(httpServer *http.Server) {
				log.Println("shutdown api server ...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := httpServer.Shutdown(ctx); err != nil {
					log.Printf("http server shutdown error:%v\n", err)
				}
			}(httpServer)

			// gracefully shutdown
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit
		},
	}

	cmdAPI.Flags().StringVarP(&serverConfigFile, "config", "c", "./local.toml", "Path to Config File")
	return cmdAPI
}

func InitGinServer(ctx context.Context) (*http.Server, error) {
	router, err := GinRouter()
	if err != nil {
		return nil, err
	}

	port := "8080"
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  viper.GetDuration("http.read_timeout"),
		WriteTimeout: viper.GetDuration("http.write_timeout"),
	}

	go func() {
		log.Printf("Server is running and listening port: %s", port)
		// service connections
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	return httpServer, nil
}

func initTracer() *tracesdk.TracerProvider {
	exporter, err := otelexporter.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		log.Fatal(err)
	}
	tp := tracesdk.NewTracerProvider(
		// sampler rate with builtin sampler
		// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk.md#built-in-samplers
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// tracesdk.WithIDGenerator(customTracer.GetCIDIDGenerator()),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("otel-example"),
			attribute.String("environment", "staging"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func GinRouter() (*gin.Engine, error) {
	gin.SetMode("debug")
	router := gin.New()

	// Init root router group
	rootGroup := router.Group("")
	// rootGroup.Use(otelgin.Middleware("my-server"))
	rootGroup.Use(customMiddleware.Middleware("my-server"))

	// general service for debugging
	rootGroup.GET("/health", health)
	rootGroup.GET("/users/:id", getUser)

	return router, nil
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	handlerName := c.HandlerName()

	// Pass the built-in `context.Context` object from http.Request to OpenTelemetry APIs
	// where required. It is available from gin.Context.Request.Context()
	_, span := tracer.Start(c.Request.Context(), handlerName, oteltrace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	c.JSON(http.StatusOK, gin.H{"status": id})
}
