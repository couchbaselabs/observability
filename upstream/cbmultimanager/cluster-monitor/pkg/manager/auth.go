// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// createJWTToken generates an auth token for the given user.
//
// NOTE: at the moment there is no way of force expiring tokens apart from restarting the service. Also the token is
// based on symmetric encryption as cbmultimanager is not a clustered product, if we ever cluster we will have to change
// to public key encryption.
func (m *Manager) createJWTToken(user string, lifeSpan time.Duration) (string, error) {
	// USE AES 256 to encrypt the tokens
	encrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{
		Algorithm: jose.A256GCMKW,
		Key:       m.config.EncryptKey,
	}, (&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"))
	if err != nil {
		return "", fmt.Errorf("could not create encrypter: %w", err)
	}

	// expire after an hour
	expiry := jwt.NumericDate(time.Now().Add(lifeSpan).Unix())
	issued := jwt.NumericDate(time.Now().Unix())

	claim := jwt.Claims{
		Issuer:    m.config.UUID,
		Subject:   user,
		Expiry:    &expiry,
		IssuedAt:  &issued,
		NotBefore: &issued,
	}

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       m.config.SignKey,
	}, (&jose.SignerOptions{}).WithType("JWT").WithContentType("JWT"))
	if err != nil {
		return "", fmt.Errorf("could not create signer: %w", err)
	}

	raw, err := jwt.SignedAndEncrypted(signer, encrypter).Claims(claim).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("could not produce token: %w", err)
	}

	return raw, err
}
