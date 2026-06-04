package config

import (
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/do/v2"
)

type JwtConfig struct {
	signingKey  []byte
	validateKey []byte
	method      jwt.SigningMethod
}

func (c *JwtConfig) GetSigningKey() []byte {
	return c.signingKey
}

func (c *JwtConfig) GetValidateKey() []byte {
	return c.validateKey
}

func (c *JwtConfig) GetMethod() jwt.SigningMethod {
	return c.method
}

func newJwtConfig(i do.Injector) (*JwtConfig, error) {

	key := os.Getenv("JWT_KEY")
	method := jwt.SigningMethodHS256

	return &JwtConfig{
		signingKey:  []byte(key),
		validateKey: []byte(key),
		method:      method,
	}, nil
}

func RegisterJwtConfig(i do.Injector) {
	do.Provide(i, newJwtConfig)
}
