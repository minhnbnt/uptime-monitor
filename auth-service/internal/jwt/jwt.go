package jwt

import (
	"maps"

	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
)

type Provider struct {
	config *config.JwtConfig
}

func RegisterProvider(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Provider, error) {
		config := do.MustInvoke[*config.JwtConfig](i)
		return &Provider{config: config}, nil
	})
}

func (j *Provider) Parse(token string) (*Token, error) {
	return j.parseWithOptions(token)
}

func (j *Provider) ParseWithIssuer(token string, expectedIssuer string) (*Token, error) {
	return j.parseWithOptions(token, jwt.WithIssuer(expectedIssuer))
}

func (j *Provider) parseWithOptions(token string, extraOpts ...jwt.ParserOption) (*Token, error) {
	keyFunc := func(t *jwt.Token) (any, error) {
		return j.config.GetValidateKey(), nil
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{j.config.GetMethod().Alg()}),
	}
	options = append(options, extraOpts...)

	parsedToken, err := jwt.Parse(token, keyFunc, options...)
	if err != nil {
		return nil, err
	}

	return &Token{token: parsedToken}, nil
}

func (j *Provider) Validate(token string) (string, error) {

	t, err := j.Parse(token)
	if err != nil {
		return "", err
	}

	return t.Issuer()
}

func (j *Provider) NewToken(issuer string, otherClaims map[string]any) (string, error) {

	claim := jwt.MapClaims{"iss": issuer}
	maps.Copy(claim, otherClaims)

	token := jwt.NewWithClaims(j.config.GetMethod(), claim)

	tokenString, err := token.SignedString(j.config.GetSigningKey())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
