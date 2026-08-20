package crypto

import (
	"bytes"
	"testing"

	"filippo.io/age"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	pub := id.Recipient().String()
	priv := id.String()

	plaintext := []byte("A0B1C2\nD3E4F5\n# comment\nN12345\n")

	ct, err := Encrypt(plaintext, pub)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}
	if bytes.Contains(ct, plaintext) {
		t.Fatal("ciphertext contains the plaintext")
	}

	got, err := Decrypt(ct, priv)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch:\n got  %q\n want %q", got, plaintext)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	server, _ := age.GenerateX25519Identity()
	attacker, _ := age.GenerateX25519Identity()

	ct, err := Encrypt([]byte("classified"), server.Recipient().String())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := Decrypt(ct, attacker.String()); err == nil {
		t.Fatal("decrypt with the wrong private key should fail")
	}
}
