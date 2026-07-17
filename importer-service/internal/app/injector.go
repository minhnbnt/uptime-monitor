package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/infrastructure/excel"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/service"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){

		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterServerClient,

		excel.RegisterExcelExporter,
		excel.RegisterExcelParser,

		service.RegisterImportService,

		handler.RegisterImportHandler,
	}

	for _, p := range packages {
		p(injector)
	}
}
