package shared

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func HandleAuth(headers map[string]string, jwksURL string) error {
	// Extract the Authorization header
	authHeader, ok := headers["Authorization"]
	if !ok {
		return errors.New("missing Authorization header")
	}

	keySet, err := jwk.Fetch(context.TODO(), jwksURL)

	_, err = jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found")
		}

		// Find the key with the specified key ID (kid) in the JWKS
		keys, found := keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key with specified kid is not present in JWKS")
		}

		var publickey interface{}
		err = keys.Raw(&publickey)
		if err != nil {
			return nil, fmt.Errorf("could not parse pubkey")
		}
		return publickey, nil
	})

	if err != nil {
		return err
	}

	return nil
}
