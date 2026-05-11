package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"httpaas/config"
	"httpaas/infra"
	"httpaas/store"
)

const maxUploadBytes = 50 << 20

var hostRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type API struct {
	cfg   config.Config
	store *store.Store
	vbox  *infra.VBox
	ssh   *infra.SSHClient
}

type errorResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type okHostResponse struct {
	OK   bool   `json:"ok"`
	Host string `json:"host"`
	IP   string `json:"ip"`
}

type okInstanceResponse struct {
	OK       bool           `json:"ok"`
	Instance store.Instance `json:"instance"`
}

func New(cfg config.Config, st *store.Store, vbox *infra.VBox, ssh *infra.SSHClient) *API {
	return &API{
		cfg:   cfg,
		store: st,
		vbox:  vbox,
		ssh:   ssh,
	}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.Handle("/api/instancias", http.HandlerFunc(a.ListInstancias))
	mux.Handle("/api/instancias/host", http.HandlerFunc(a.AceptarHost))
	mux.Handle("/api/provisionar", http.HandlerFunc(a.Publicar))
	mux.Handle("/api/instancias/publicar", http.HandlerFunc(a.Publicar))
	mux.Handle("/api/eliminar/", http.HandlerFunc(a.Eliminar))
	mux.Handle("/api/instancias/", http.HandlerFunc(a.Eliminar))
}

func (a *API) ListInstancias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{OK: false, Error: "metodo no permitido"})
		return
	}

	list := a.store.List()
	writeJSON(w, http.StatusOK, list)
}

func (a *API) AceptarHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{OK: false, Error: "metodo no permitido"})
		return
	}

	var payload struct {
		Host string `json:"host"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "json invalido"})
		return
	}

	host := strings.TrimSpace(payload.Host)
	if !isValidHost(host) {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "hostname invalido"})
		return
	}

	if a.store.Exists(host) {
		writeJSON(w, http.StatusConflict, errorResponse{OK: false, Error: "hostname ya existe"})
		return
	}

	ip, err := a.store.NextIP(a.cfg.BaseIP, a.cfg.StartOctet)
	if err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, okHostResponse{OK: true, Host: host, IP: ip})
}

func (a *API) Publicar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{OK: false, Error: "metodo no permitido"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "archivo demasiado grande o formulario invalido"})
		return
	}

	host := strings.TrimSpace(r.FormValue("host"))
	if !isValidHost(host) {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "hostname invalido"})
		return
	}

	if a.store.Exists(host) {
		writeJSON(w, http.StatusConflict, errorResponse{OK: false, Error: "hostname ya existe"})
		return
	}

	file, header, err := r.FormFile("archivo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "archivo requerido"})
		return
	}
	defer file.Close()

	if err := os.MkdirAll(a.cfg.UploadTmpDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "no se pudo crear carpeta temporal"})
		return
	}

	timestamp := time.Now().UnixNano()
	tmpName := fmt.Sprintf("%s_%d_%s", host, timestamp, filepath.Base(header.Filename))
	tmpPath := filepath.Join(a.cfg.UploadTmpDir, tmpName)

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "no se pudo guardar archivo"})
		return
	}

	if _, err := io.Copy(tmpFile, file); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "no se pudo escribir archivo"})
		return
	}
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	ip, err := a.store.NextIP(a.cfg.BaseIP, a.cfg.StartOctet)
	if err != nil {
		writeJSON(w, http.StatusConflict, errorResponse{OK: false, Error: err.Error()})
		return
	}

	createdVM := false
	rollback := func() {
		if !createdVM {
			return
		}
		state, err := a.vbox.MachineState(host)
		if err != nil {
			log.Printf("rollback state error: %v", err)
		} else {
			log.Printf("rollback: estado actual de %s = %s", host, state)
		}

		if err := a.vbox.PowerOff(host); err != nil {
			log.Printf("rollback poweroff: %v", err)
		}
		// Required delay before unregister to avoid locked VM state.
		time.Sleep(10 * time.Second)

		if err := a.vbox.WaitUntilUnlocked(host, 60*time.Second, 2*time.Second); err != nil {
			log.Printf("rollback wait unlocked: %v", err)
		}
		// Try unregistering multiple times in case VBoxManage reports the VM is locked.
		if err := a.vbox.UnregisterDeleteRetry(host, 6, 5*time.Second); err != nil {
			log.Printf("rollback unregister (after retries): %v", err)
		}
	}

	log.Printf("[PROVISIONAR %s] 1. clonando VM desde plantilla %s (snapshot %s)", host, a.cfg.TemplateName, a.cfg.SnapshotName)
	if err := a.vbox.CloneLinkedVM(a.cfg.TemplateName, a.cfg.SnapshotName, host); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en CloneLinkedVM: %v", host, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error clonando VM"})
		return
	}
	createdVM = true
	log.Printf("[PROVISIONAR %s] OK clonada", host)

	log.Printf("[PROVISIONAR %s] 2. configurando memoria y adaptador de red", host)
	if err := a.vbox.ModifyVM(host,
		"--memory", "512",
		"--cpus", "1",
		"--nic1", "hostonly",
		"--hostonlyadapter1", a.cfg.HostOnlyAdapter,
	); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en ModifyVM: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error configurando VM"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK configurada", host)

	log.Printf("[PROVISIONAR %s] 3. iniciando VM", host)
	if err := a.vbox.StartVM(host); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en StartVM: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error iniciando VM"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK iniciada", host)

	log.Printf("[PROVISIONAR %s] 4. esperando Guest Additions (hasta 5m, cada 5s)", host)
	if err := a.vbox.WaitForGuestAdditions(host, 5*time.Minute, 5*time.Second); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en WaitForGuestAdditions: %v", host, err)
		rollback()
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{OK: false, Error: "guest additions no disponibles"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK Guest Additions activas", host)

	log.Printf("[PROVISIONAR %s] 5. esperando guestcontrol listo (hasta 5m, cada 5s)", host)
	if err := a.vbox.WaitForGuestControlReady(host, "root", "root", 5*time.Minute, 5*time.Second); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en WaitForGuestControlReady: %v", host, err)
		rollback()
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{OK: false, Error: "guestcontrol no disponible"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK guestcontrol listo", host)

	log.Printf("[PROVISIONAR %s] 6. configurando IP estática via guestcontrol (%s)", host, ip)
	if err := a.vbox.ConfigureStaticIPViaGuestControlWithRetry(host, "root", "root", "enp0s3", ip, "255.255.255.0", 12, 5*time.Second); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en ConfigureStaticIPViaGuestControl: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error configurando red por guestcontrol"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK IP estática configurada por guestcontrol", host)

	log.Printf("[PROVISIONAR %s] 7. esperando 10s para aplicar red antes de SSH", host)
	time.Sleep(10 * time.Second)

	waitTimeout := time.Duration(a.cfg.SSHWaitTimeoutSeconds) * time.Second
	waitInterval := time.Duration(a.cfg.SSHRetrySeconds) * time.Second
	log.Printf("[PROVISIONAR %s] 8. probando SSH en %s (timeout %s, interval %s)", host, ip, waitTimeout, waitInterval)
	if err := a.ssh.WaitForSSH(ip, waitTimeout, waitInterval); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en WaitForSSH: %v", host, err)
		rollback()
		writeJSON(w, http.StatusGatewayTimeout, errorResponse{OK: false, Error: "SSH no disponible"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK SSH disponible", host)

	log.Printf("[PROVISIONAR %s] 9. configurando hostname", host)
	if err := infra.ConfigureHostname(a.ssh, ip, host); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en ConfigureHostname: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error configurando hostname"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK hostname configurado", host)

	log.Printf("[PROVISIONAR %s] 10. registrando en DNS via SSH en vm-ns1 (%s)", host, a.cfg.DNSServerIP)
	if err := infra.AddDNS(a.ssh, a.cfg.DNSServerIP, a.cfg.Domain, host, ip); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en AddDNS: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error registrando DNS"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK DNS registrado", host)

	log.Printf("[PROVISIONAR %s] 11. descomprimiendo y copiando contenido web", host)
	if err := infra.DeployContent(a.ssh, ip, host, tmpPath); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en DeployContent: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error desplegando contenido"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK contenido desplegado", host)

	log.Printf("[PROVISIONAR %s] 12. configurando Apache VirtualHost", host)
	if err := infra.ConfigureVirtualHost(a.ssh, ip, host, a.cfg.Domain); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en ConfigureVirtualHost: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error configurando apache"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK VirtualHost configurado", host)

	log.Printf("[PROVISIONAR %s] 13. configurando networking estática", host)
	if err := infra.ConfigureStaticNetworking(a.ssh, ip, host, ip, a.cfg.DNSServerIP); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en ConfigureStaticNetworking: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error configurando red estática"})
		return
	}
	log.Printf("[PROVISIONAR %s] OK networking configurado", host)

	log.Printf("[PROVISIONAR %s] 14. guardando en store", host)
	inst := store.Instance{
		Host:    host,
		IP:      fmt.Sprintf("%s/24", ip),
		Domain:  fmt.Sprintf("http://%s.%s", host, a.cfg.Domain),
		Created: time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := a.store.Add(inst); err != nil {
		log.Printf("[PROVISIONAR %s] ERROR en store.Add: %v", host, err)
		rollback()
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error guardando instancia"})
		return
	}
	log.Printf("[PROVISIONAR %s] ✓ ÉXITO - instancia completada", host)

	writeJSON(w, http.StatusOK, okInstanceResponse{OK: true, Instance: inst})
}

func (a *API) Eliminar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{OK: false, Error: "metodo no permitido"})
		return
	}

	host := strings.TrimPrefix(r.URL.Path, "/api/eliminar/")
	host = strings.TrimPrefix(host, "/api/instancias/")
	host = strings.TrimSpace(host)
	if !isValidHost(host) {
		writeJSON(w, http.StatusBadRequest, errorResponse{OK: false, Error: "hostname invalido"})
		return
	}

	inst, ok := a.store.Find(host)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{OK: false, Error: "instancia no encontrada"})
		return
	}

	if err := a.vbox.PowerOff(host); err != nil {
		log.Printf("poweroff error: %v", err)
	}
	if err := a.vbox.UnregisterDelete(host); err != nil {
		log.Printf("unregister error: %v", err)
	}
	if err := infra.DeleteDNS(a.ssh, a.cfg.DNSServerIP, a.cfg.Domain, host); err != nil {
		log.Printf("dns delete error: %v", err)
	}

	if _, _, err := a.store.Delete(host); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{OK: false, Error: "error eliminando instancia"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "host": inst.Host})
}

func isValidHost(host string) bool {
	if host == "" || strings.Contains(host, ".") {
		return false
	}
	return hostRegex.MatchString(host)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode error: %v", err)
	}
}
