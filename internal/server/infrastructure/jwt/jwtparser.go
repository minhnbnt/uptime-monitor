package jwt

import (
	"maps"

	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

type JwtParser struct {
	config *config.JwtConfig
}

func RegisterJwtParser(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*JwtParser, error) {
		config := do.MustInvoke[*config.JwtConfig](i)
		return &JwtParser{config: config}, nil
	})
}

func (j *JwtParser) Parse(token string) (*Token, error) {

	keyFunc := func(t *jwt.Token) (any, error) {
		return j.config.GetValidateKey(), nil
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{j.config.GetMethod().Alg()}),
	}

	parsedToken, err := jwt.Parse(token, keyFunc, options...)
	if err != nil {
		return nil, err
	}

	return &Token{token: parsedToken}, nil
}

func (j *JwtParser) Validate(token string) (string, error) {

	t, err := j.Parse(token)
	if err != nil {
		return "", err
	}

	return t.Issuer()
}

func (j *JwtParser) NewToken(issuer string, otherClaims map[string]any) (string, error) {

	claim := jwt.MapClaims{"iss": issuer}
	maps.Copy(claim, otherClaims)

	token := jwt.NewWithClaims(j.config.GetMethod(), claim)

	tokenString, err := token.SignedString(j.config.GetSigningKey())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
