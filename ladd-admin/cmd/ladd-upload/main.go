package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/crypto"
	"github.com/chasehaye/nas-pipeline/ladd-admin/internal/api"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2:])
	case "upload":
		upload(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  ladd-upload keygen [--out DIR]")
	fmt.Fprintln(os.Stderr, "  ladd-upload upload --file <ladd.txt> --url <endpoint> \\")
	fmt.Fprintln(os.Stderr, "    (--recipient <age1..> | --recipient-file server-recipient.pub) \\")
	fmt.Fprintln(os.Stderr, "    (--sign-key <b64> | --sign-key-file operator-signing.key)")
	os.Exit(2)
}

func keygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", ".", "directory to write key files")
	_ = fs.Parse(args)

	signPriv, signPub, err := crypto.GenerateSigningKeypair()
	if err != nil {
		fatalf("generate signing keypair: %v", err)
	}
	identity, recipient, err := crypto.GenerateEncryptionKeypair()
	if err != nil {
		fatalf("generate encryption keypair: %v", err)
	}

	write := func(name, content string, mode os.FileMode) string {
		p := filepath.Join(*out, name)
		if err := os.WriteFile(p, []byte(content+"\n"), mode); err != nil {
			fatalf("write %s: %v", p, err)
		}
		return p
	}

	opSign := write("operator-signing.key", signPriv, 0o600)
	opPub := write("operator-signing.pub", signPub, 0o644)
	srvID := write("server-identity.txt", identity, 0o600)
	srvPub := write("server-recipient.pub", recipient, 0o644)

	fmt.Println("generated key material:")
	fmt.Printf("  %-26s  keep on your machine  (upload --sign-key-file)\n", opSign)
	fmt.Printf("  %-26s  keep on your machine  (upload --recipient-file)\n", srvPub)
	fmt.Printf("  %-26s  put in the cluster Secret as operator.pub\n", opPub)
	fmt.Printf("  %-26s  put in the cluster Secret as identity.txt\n", srvID)
	fmt.Println()
	fmt.Println("create the Secret (server side):")
	fmt.Printf("  kubectl create secret generic ladd-admin-keys -n nas \\\n")
	fmt.Printf("    --from-file=identity.txt=%s \\\n", filepath.Base(srvID))
	fmt.Printf("    --from-file=operator.pub=%s\n", filepath.Base(opPub))
}

func upload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	file := fs.String("file", "", "path to the LADD_Industry_Filter_*_YYYYMMDD.txt file")
	recipient := fs.String("recipient", "", "server age public key (age1...)")
	recipFile := fs.String("recipient-file", "", "file with the server age public key")
	signKey := fs.String("sign-key", "", "operator signing private key (base64)")
	signKeyFile := fs.String("sign-key-file", "", "file with the operator signing private key")
	url := fs.String("url", "", "upload endpoint, e.g. https://admin.example.com/upload")
	_ = fs.Parse(args)

	if *file == "" || *url == "" {
		fatalf("--file and --url are required")
	}
	pub := valueOrFile("recipient", *recipient, *recipFile, firstLinePrefixed("age1"))
	priv := valueOrFile("sign-key", *signKey, *signKeyFile, strings.TrimSpace)

	content, err := os.ReadFile(*file)
	if err != nil {
		fatalf("read file: %v", err)
	}

	ct, err := crypto.Encrypt(content, pub)
	if err != nil {
		fatalf("encrypt: %v", err)
	}

	req := api.UploadRequest{Filename: filepath.Base(*file), Ciphertext: ct}
	sig, err := crypto.Sign(req.SigningBytes(), priv)
	if err != nil {
		fatalf("sign: %v", err)
	}
	req.Signature = sig

	body, err := json.Marshal(req)
	if err != nil {
		fatalf("marshal: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(*url, "application/json", bytes.NewReader(body))
	if err != nil {
		fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	var out api.UploadResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || !out.OK {
		fatalf("upload rejected (%d): %s", resp.StatusCode, out.Message)
	}
	fmt.Printf("uploaded %s: %d entries\n", filepath.Base(*file), out.Entries)
}

func valueOrFile(name, value, path string, pick func(string) string) string {
	switch {
	case value != "" && path != "":
		fatalf("--%s and --%s-file are mutually exclusive", name, name)
	case value != "":
		return value
	case path != "":
		b, err := os.ReadFile(path)
		if err != nil {
			fatalf("read %s file: %v", name, err)
		}
		return pick(string(b))
	default:
		fatalf("one of --%s or --%s-file is required", name, name)
	}
	return ""
}

func firstLinePrefixed(prefix string) func(string) string {
	return func(s string) string {
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				return line
			}
		}
		return strings.TrimSpace(s)
	}
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
