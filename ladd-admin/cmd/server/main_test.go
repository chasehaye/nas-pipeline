package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/api"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/config"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/crypto"
)

type mockSecrets struct {
	called   bool
	filename string
	content  []byte
}

func (m *mockSecrets) Replace(_ context.Context, filename string, content []byte) error {
	m.called = true
	m.filename = filename
	m.content = content
	return nil
}

func newTestServer(t *testing.T) (s *server, ageID *age.X25519Identity, signPriv string, mock *mockSecrets) {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("age keygen: %v", err)
	}
	priv, pub, err := crypto.GenerateSigningKeypair()
	if err != nil {
		t.Fatalf("sign keygen: %v", err)
	}
	operatorPub, err := crypto.ParseSigningPublicKey(pub)
	if err != nil {
		t.Fatalf("parse pub: %v", err)
	}
	mock = &mockSecrets{}
	s = &server{
		cfg:         config.Config{MaxAge: 9 * 24 * time.Hour, MaxUploadSize: 4 << 20},
		identity:    id.String(),
		operatorPub: operatorPub,
		secrets:     mock,
	}
	return s, id, priv, mock
}

func signedBody(t *testing.T, id *age.X25519Identity, signPriv, filename string, plain []byte) []byte {
	t.Helper()
	ct, err := crypto.Encrypt(plain, id.Recipient().String())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	req := api.UploadRequest{Filename: filename, Ciphertext: ct}
	sig, err := crypto.Sign(req.SigningBytes(), signPriv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Signature = sig
	body, _ := json.Marshal(req)
	return body
}

func post(s *server, body []byte) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.upload(rec, httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body)))
	return rec
}

func todayName() string {
	return "LADD_Industry_Filter_CUI_SP_PRVCY_" + time.Now().Format("20060102") + ".txt"
}

func TestUploadHappyPath(t *testing.T) {
	s, id, signPriv, mock := newTestServer(t)
	plain := []byte("A0B1C2\nD3E4F5\nN12345\n")

	rec := post(s, signedBody(t, id, signPriv, todayName(), plain))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !mock.called {
		t.Fatal("secret writer was never called")
	}
	if mock.filename != todayName() {
		t.Fatalf("stored filename = %q, want %q", mock.filename, todayName())
	}
	if !bytes.Equal(mock.content, plain) {
		t.Fatalf("stored content mismatch:\n got  %q\n want %q", mock.content, plain)
	}
}

func TestUploadRejectsBadSignature(t *testing.T) {
	s, id, _, mock := newTestServer(t)

	attacker, _, _ := crypto.GenerateSigningKeypair()
	ct, _ := crypto.Encrypt([]byte("A0B1C2\n"), id.Recipient().String())
	req := api.UploadRequest{Filename: todayName(), Ciphertext: ct}
	sig, _ := crypto.Sign(req.SigningBytes(), attacker)
	req.Signature = sig
	body, _ := json.Marshal(req)

	rec := post(s, body)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if mock.called {
		t.Fatal("secret writer must NOT be called for a bad signature")
	}
}

func TestUploadRejectsGarbageCiphertext(t *testing.T) {
	s, _, signPriv, mock := newTestServer(t)

	req := api.UploadRequest{Filename: todayName(), Ciphertext: []byte("not age ciphertext")}
	sig, _ := crypto.Sign(req.SigningBytes(), signPriv)
	req.Signature = sig
	body, _ := json.Marshal(req)

	rec := post(s, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if mock.called {
		t.Fatal("secret writer must NOT be called when decryption fails")
	}
}

func TestUploadRejectsStaleFile(t *testing.T) {
	s, id, signPriv, mock := newTestServer(t)
	stale := time.Now().AddDate(0, 0, -30).Format("20060102")
	name := "LADD_Industry_Filter_CUI_SP_PRVCY_" + stale + ".txt"

	rec := post(s, signedBody(t, id, signPriv, name, []byte("A0B1C2\n")))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if mock.called {
		t.Fatal("secret writer must NOT be called for a stale file (fail-closed)")
	}
}
