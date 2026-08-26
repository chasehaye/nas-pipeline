package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/chasehaye/nas-pipeline/platform/health"
	"github.com/chasehaye/nas-pipeline/platform/log"

	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/api"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/config"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/crypto"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/ksecret"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/validate"
)

type secretWriter interface {
	Replace(ctx context.Context, filename string, content []byte) error
}

type server struct {
	cfg         config.Config
	identity    string
	operatorPub ed25519.PublicKey
	secrets     secretWriter
}

func main() {
	// Shared platform: JSON structured logging as the process-wide default.
	log.Init(os.Getenv("LOG_LEVEL"))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file loaded; using environment and defaults", "err", err)
	}

	cfg := config.Load()

	identity, err := readIdentity(cfg.IdentityPath)
	if err != nil {
		slog.Error("load identity failed", "err", err)
		os.Exit(1)
	}

	operatorPub, err := readOperatorPub(cfg.OperatorPubKey)
	if err != nil {
		slog.Error("load operator public key failed", "err", err)
		os.Exit(1)
	}

	secrets, err := ksecret.New(cfg.SecretNS, cfg.SecretName)
	if err != nil {
		slog.Error("k8s client init failed", "err", err)
		os.Exit(1)
	}

	s := &server{cfg: cfg, identity: identity, operatorPub: operatorPub, secrets: secrets}

	mux := http.NewServeMux()
	// Shared platform health probes: liveness always ok; readiness ok once serving.
	mux.HandleFunc("GET /healthz", health.Live)
	mux.Handle("GET /readyz", health.Ready())
	mux.HandleFunc("POST /upload", s.upload)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("ladd-admin listening", "addr", cfg.Addr, "secret", cfg.SecretNS+"/"+cfg.SecretName)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

func (s *server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadSize)

	var req api.UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, api.UploadResponse{Message: "invalid request body"})
		return
	}

	if err := crypto.Verify(req.SigningBytes(), req.Signature, s.operatorPub); err != nil {
		writeJSON(w, http.StatusUnauthorized, api.UploadResponse{Message: "signature verification failed"})
		return
	}

	plain, err := crypto.Decrypt(req.Ciphertext, s.identity)
	if err != nil {
		slog.Warn("decrypt failed", "err", err)
		writeJSON(w, http.StatusBadRequest, api.UploadResponse{Message: "decryption failed"})
		return
	}

	res, err := validate.Check(req.Filename, plain, s.cfg.MaxAge)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, api.UploadResponse{Message: err.Error()})
		return
	}

	if err := s.secrets.Replace(r.Context(), res.Name, plain); err != nil {
		slog.Error("secret update failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, api.UploadResponse{Message: "failed to store list"})
		return
	}

	slog.Info("LADD updated", "name", res.Name, "entries", res.Entries, "effective", res.Date.Format("2006-01-02"))
	writeJSON(w, http.StatusOK, api.UploadResponse{OK: true, Entries: res.Entries, Message: "updated"})
}

func writeJSON(w http.ResponseWriter, status int, body api.UploadResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func readIdentity(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return line, nil
		}
	}
	return "", errors.New("no AGE-SECRET-KEY line in identity file")
}

func readOperatorPub(path string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return crypto.ParseSigningPublicKey(string(b))
}
