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
)

var (
	quit      = make(chan os.Signal, 5)
	systemCtx context.Context
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

	port := viper.GetString("http.port")
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

func GinRouter() (*gin.Engine, error) {
	gin.SetMode(viper.GetString("http.mode"))
	router := gin.New()

	// Init root router group
	rootGroup := router.Group("")

	// general service for debugging
	rootGroup.GET("/health", health)

	return router, nil
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
