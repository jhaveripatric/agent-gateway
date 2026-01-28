package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// JWTVerifier validates JWTs using public keys.
type JWTVerifier struct {
	publicKeys map[string]*ecdsa.PublicKey // kid -> key
	issuer     string
	audience   string
}

// Claims represents JWT claims with user info.
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewJWTVerifier creates a new JWT verifier.
func NewJWTVerifier(issuer, audience string) *JWTVerifier {
	return &JWTVerifier{
		publicKeys: make(map[string]*ecdsa.PublicKey),
		issuer:     issuer,
		audience:   audience,
	}
}

// LoadPublicKey loads a public key from a PEM file.
func (v *JWTVerifier) LoadPublicKey(keyID, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("no PEM block found")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an ECDSA public key")
	}

	v.publicKeys[keyID] = ecdsaPub
	return nil
}

// HasKeys returns true if any public keys are loaded.
func (v *JWTVerifier) HasKeys() bool {
	return len(v.publicKeys) > 0
}

// Verify validates a JWT and returns claims.
func (v *JWTVerifier) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify algorithm is ES256
		if token.Method.Alg() != "ES256" {
			return nil, fmt.Errorf("unexpected algorithm: %s", token.Method.Alg())
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in header")
		}

		// Lookup public key
		key, ok := v.publicKeys[kid]
		if !ok {
			return nil, fmt.Errorf("unknown kid: %s", kid)
		}

		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Validate issuer
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Validate audience
	hasAudience := false
	for _, aud := range claims.Audience {
		if aud == v.audience {
			hasAudience = true
			break
		}
	}
	if !hasAudience {
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}
