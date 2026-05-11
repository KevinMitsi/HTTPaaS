package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpaas/api"
	"httpaas/config"
	"httpaas/infra"
	"httpaas/store"
)

func main() {
	loadDotEnv(".env")

	cfg := config.Load()
	if cfg.SSHKeyPath == "" {
		log.Fatal("SSH_KEY_PATH es requerido")
	}

	if err := os.MkdirAll(cfg.UploadTmpDir, 0o755); err != nil {
		log.Fatalf("no se pudo crear UploadTmpDir: %v", err)
	}

	st, err := store.New(cfg.StorePath)
	if err != nil {
		log.Fatalf("error leyendo store: %v", err)
	}

	sshClient, err := infra.NewSSHClient(cfg.SSHUser, cfg.SSHKeyPath, cfg.SSHPort, time.Duration(cfg.SSHTimeoutSeconds)*time.Second)
	if err != nil {
		log.Fatalf("error configurando SSH: %v", err)
	}

	vbox := infra.NewVBox(cfg.VBoxManagePath)
	apiServer := api.New(cfg, st, vbox, sshClient)

	// Ensure DNS server is running before starting HTTPaaS
	log.Println("[STARTUP] Verificando servidor DNS local (vm-ns1)...")
	if err := vbox.EnsureDNSServerRunning("vm-ns1"); err != nil {
		log.Fatalf("error iniciando vm-ns1: %v", err)
	}

	mux := http.NewServeMux()
	apiServer.Register(mux)

	staticPath := "./static"
	if abs, err := filepath.Abs(staticPath); err == nil {
		staticPath = abs
	}

	if _, err := os.Stat(staticPath); err != nil {
		log.Printf("static dir no encontrado: %s", staticPath)
	}

	mux.Handle("/", http.FileServer(http.Dir(staticPath)))

	handler := api.WithLogging(api.WithCORS(mux))
	server := &http.Server{
		Addr:              cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("HTTPaaS escuchando en %s", cfg.Port)
	log.Fatal(server.ListenAndServe())
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
