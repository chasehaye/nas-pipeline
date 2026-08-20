package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"filippo.io/age"
)

func GenerateSigningKeypair() (privB64, pubB64 string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub), nil
}

func Sign(data []byte, privB64 string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privB64))
	if err != nil {
		return nil, fmt.Errorf("decode signing key: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("signing key wrong size (%d, want %d)", len(b), ed25519.PrivateKeySize)
	}
	return ed25519.Sign(ed25519.PrivateKey(b), data), nil
}

func ParseSigningPublicKey(pubB64 string) (ed25519.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pubB64))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key wrong size (%d, want %d)", len(b), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(b), nil
}

func Verify(data, sig []byte, pub ed25519.PublicKey) error {
	if !ed25519.Verify(pub, data, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

func GenerateEncryptionKeypair() (identity, recipient string, err error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	return id.String(), id.Recipient().String(), nil
}
