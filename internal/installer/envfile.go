package installer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadDotEnv(root string) {
	path := filepath.Join(root, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		os.Setenv(key, strings.TrimSpace(val))
	}
}

func ensureEnvFile(root string) error {
	envPath := filepath.Join(root, ".env")
	examplePath := filepath.Join(root, ".env.example")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		data, readErr := os.ReadFile(examplePath)
		if readErr != nil {
			return fmt.Errorf("missing .env and .env.example: %w", readErr)
		}
		if err := os.WriteFile(envPath, data, 0644); err != nil {
			return err
		}
	}
	return ensureBootstrapToken(envPath)
}

func ensureBootstrapToken(envPath string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	empty := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "INSTALL_BOOTSTRAP_TOKEN=") {
			found = true
			val := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "INSTALL_BOOTSTRAP_TOKEN="))
			if val == "" {
				empty = true
				token, genErr := randomToken()
				if genErr != nil {
					return genErr
				}
				lines[i] = "INSTALL_BOOTSTRAP_TOKEN=" + token
			}
			break
		}
	}
	if !found {
		token, genErr := randomToken()
		if genErr != nil {
			return genErr
		}
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "INSTALL_BOOTSTRAP_TOKEN="+token)
	}
	if !found || empty {
		out := strings.Join(lines, "\n")
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		return os.WriteFile(envPath, []byte(out), 0644)
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func managementPort() string {
	if v := strings.TrimSpace(os.Getenv("MANAGEMENT_PORT")); v != "" {
		return v
	}
	return "8188"
}

func managementBaseURL() string {
	return "http://127.0.0.1:" + managementPort()
}
