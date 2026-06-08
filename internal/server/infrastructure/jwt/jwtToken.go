package jwt

import (
	"errors"
	"fmt"

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
