package main

//go:generate go tool oapi-codegen -config=oapi-codegen.yml api/spec.yaml

import (
	"net/http"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/handler"
	"github.com/minhnbnt/uptime-monitor/internal/infrastructure/logger"
	repo "github.com/minhnbnt/uptime-monitor/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor/internal/service"
)

func main() {

	injector := do.New(
		config.RegisterZapLogger,
		config.RegisterGORMDB,
		logger.RegisterLogger,
		repo.RegisterServerRepository,
		service.RegisterServerService,
		handler.RegisterMockServer,
	)

	router := gin.Default()

	logger := do.MustInvoke[*zap.Logger](injector)

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/api/v1/hello", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "Hello, world!"})
	})

	server := do.MustInvoke[*handler.MockServer](injector)
	api.RegisterHandlers(router, server)

	http.ListenAndServe(":8080", router)
}
