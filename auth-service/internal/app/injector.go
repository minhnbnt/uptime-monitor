package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/argon2"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/jwt"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/token"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/service"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/handler"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	config.RegisterConfigPath(configPath)(injector)
	config.RegisterLogger(dev)(injector)
	config.RegisterJwtConfig(injector)
	config.RegisterTokenConfig(injector)
	config.RegisterArgon2Config(injector)
	config.RegisterGORMDB(injector)
	config.RegisterRedisClient(injector)

	repository.RegisterUserRepository(injector)
	repository.RegisterRedisRevokedTokenRepository(injector)

	jwt.RegisterProvider(injector)
	argon2.RegisterArgon2PasswordEncoder(injector)
	token.RegisterTokenGenerator(injector)
	token.RegisterTokenValidator(injector)

	service.RegisterAuthService(injector)

	handler.RegisterAuthHandler(injector)
}
