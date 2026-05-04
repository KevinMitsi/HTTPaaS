package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"httpaas/api"
	"httpaas/config"
	"httpaas/infra"
	"httpaas/store"
)

func main() {
	cfg := config.Load()
	if cfg.SSHKeyPath == "" {
		log.Fatal("HTTPAAS_SSH_KEY es requerido")
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
