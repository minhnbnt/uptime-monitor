package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/argon2"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/jwt"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/service"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){

		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterJwtConfig,
		config.RegisterTokenConfig,
		config.RegisterArgon2Config,
		config.RegisterGORMDB,
		config.RegisterRedisClient,

		repository.RegisterUserRepository,
		repository.RegisterRedisRevokedTokenRepository,

		jwt.RegisterProvider,
		argon2.RegisterArgon2PasswordEncoder,
		token.RegisterTokenGenerator,
		token.RegisterTokenValidator,

		service.RegisterAuthService,

		handler.RegisterAuthHandler,
		handler.RegisterForwardAuthHandler,
	}

	for _, p := range packages {
		p(injector)
	}
}
