package main

//go:generate go tool oapi-codegen -config=oapi-codegen.yml api/spec.yaml

import (
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
	monitorhandler "github.com/minhnbnt/uptime-monitor/internal/monitor/handler"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
	"github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/logger"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

func main() {

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	injector := do.New(

		config.RegisterZapLogger,
		config.RegisterGORMDB,
		config.RegisterTemporalConfig,
		config.RegisterTemporalClient,

		logger.RegisterLogger,
		repo.RegisterServerRepository,
		repo.RegisterEndpointRepository,
		repo.RegisterPingSchedulerRepository,

		infra.RegisterPingWorker,

		service.RegisterServerService,
		service.RegisterEndpointService,
		handler.RegisterRequestValidator,
		handler.RegisterServerHandler,
		handler.RegisterEndpointHandler,
		server.RegisterCompositeHandler,

		monitorhandler.RegisterTemporalWorkerRunner,
	)

	waitgroup.Go(func() { injector.ShutdownOnSignals(syscall.SIGTERM) })

	waitgroup.Go(func() {
		runner := do.MustInvoke[*monitorhandler.TemporalWorkerRunner](injector)
		runner.RunTemporalWorker()
	})

	waitgroup.Go(func() {

		router := gin.Default()
		router.Use(cors.Default())

		logger := do.MustInvoke[*zap.Logger](injector)

		router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
		router.Use(ginzap.RecoveryWithZap(logger, true))

		router.GET("/api/v1/hello", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"message": "Hello, world!"})
		})

		server := do.MustInvoke[*server.CompositeHandler](injector)
		api.RegisterHandlers(router, server)

		http.ListenAndServe(":8080", router)
	})
}
