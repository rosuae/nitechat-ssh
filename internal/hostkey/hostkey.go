package hostkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh"
)

const (
	hostKeyDirectory = "cmd/nite-chat/.ssh"
	hostKeyFilename  = "host_ed25519"
)

func DefaultPath() (string, error) {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("find hostkey package path")
	}

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", ".."))
	return filepath.Join(projectRoot, hostKeyDirectory, hostKeyFilename), nil
}

func LoadSigner(path string) (ssh.Signer, error) {
	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read host key: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, fmt.Errorf("create host key directory: %w", err)
		}

		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate host key: %w", err)
		}

		privateKeyBytes = privateKey
		if err := os.WriteFile(path, privateKeyBytes, 0600); err != nil {
			return nil, fmt.Errorf("write host key: %w", err)
		}
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid Ed25519 private key size")
	}

	signer, err := ssh.NewSignerFromKey(ed25519.PrivateKey(privateKeyBytes))
	if err != nil {
		return nil, fmt.Errorf("create host key signer: %w", err)
	}

	return signer, nil
}
