package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/ontimeclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/serverclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/userclient"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/service"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){
		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterTemporalClient,
		config.RegisterMailClient,
		config.RegisterGRPCClient,
		config.RegisterGRPCOntimeClient,

		repository.RegisterNotificationConfigRepository,

		userclient.RegisterClient,
		serverclient.RegisterClient,
		ontimeclient.RegisterClient,

		infrastructure.RegisterMailer,
		infrastructure.RegisterDigestStarter,

		service.RegisterNotificationService,
		service.RegisterDigestService,

		handler.RegisterNotificationHandler,

		handler.RegisterDigestWorkerRunner,
	}

	for _, p := range packages {
		p(injector)
	}
}
