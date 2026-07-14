package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Token struct {
	token *jwt.Token
}

func (t *Token) Claims() (jwt.MapClaims, error) {

	claims, ok := t.token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

func (t *Token) getClaimsField(field string) (any, error) {

	claims, err := t.Claims()
	if err != nil {
		return nil, err
	}

	value, ok := claims[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found", field)
	}

	return value, nil
}

func (t *Token) Subject() (string, error) {

	sub, err := t.getClaimsField("sub")
	if err != nil {
		return "", err
	}

	subject, ok := sub.(string)
	if !ok {
		return "", errors.New("invalid subject")
	}

	return subject, nil
}

func (t *Token) Issuer() (string, error) {

	iss, err := t.getClaimsField("iss")
	if err != nil {
		return "", err
	}

	issuer, ok := iss.(string)
	if !ok {
		return "", errors.New("invalid issuer")
	}

	return issuer, nil
}

func (t *Token) JTI() (string, error) {

	jti, err := t.getClaimsField("jti")
	if err != nil {
		return "", err
	}

	id, ok := jti.(string)
	if !ok {
		return "", errors.New("invalid jti")
	}

	return id, nil
}

func (t *Token) Expiry() (time.Time, error) {

	raw, err := t.getClaimsField("exp")
	if err != nil {
		return time.Time{}, err
	}

	var unix int64
	switch v := raw.(type) {
	case float64:
		unix = int64(v)
	case int64:
		unix = v
	default:
		return time.Time{}, errors.New("invalid exp")
	}

	return time.Unix(unix, 0), nil
}
