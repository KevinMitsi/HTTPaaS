package config

import (
	"os"
	"strconv"
)

type Config struct {
	BaseIP                  string
	StartOctet              int
	Domain                  string
	SSHKeyPath              string
	SSHUser                 string
	SSHPort                 int
	DNSServerIP             string
	TemplateName            string
	VBoxManagePath          string
	HostOnlyAdapter         string
	SnapshotName            string
	Port                    string
	StorePath               string
	UploadTmpDir            string
	SSHTimeoutSeconds       int
	SSHRetrySeconds         int
	SSHWaitTimeoutSeconds   int
	InitialBootDelaySeconds int
}

func Load() Config {
	return Config{
		BaseIP:            getenv("HTTPAAS_BASE_IP", "192.168.10."),
		StartOctet:        getenvInt("HTTPAAS_START_OCTET", 30),
		Domain:            getenv("HTTPAAS_DOMAIN", "cloud.local"),
		SSHKeyPath:        getenvAny("SSH_KEY_PATH", "HTTPAAS_SSH_KEY"),
		SSHUser:           getenvAny("SSH_USER", "HTTPAAS_SSH_USER"),
		SSHPort:           getenvInt("HTTPAAS_SSH_PORT", 22),
		DNSServerIP:       getenv("HTTPAAS_DNS_IP", "192.168.10.10"),
		TemplateName:      getenv("HTTPAAS_TEMPLATE", "vm-plantilla"),
		VBoxManagePath:    getenv("HTTPAAS_VBOXMANAGE", `C:\Program Files\Oracle\VirtualBox\VBoxManage.exe`),
		HostOnlyAdapter:   getenv("HTTPAAS_HOSTONLY_ADAPTER", "Ethernet 3"),
		SnapshotName:      getenv("HTTPAAS_SNAPSHOT_NAME", "base"),
		Port:              getenv("HTTPAAS_PORT", ":8080"),
		StorePath:         getenv("HTTPAAS_STORE", "./instancias.json"),
		UploadTmpDir:      getenv("HTTPAAS_UPLOAD_TMP", "./uploads"),
		SSHTimeoutSeconds: getenvInt("HTTPAAS_SSH_TIMEOUT", 30),
		SSHRetrySeconds:   getenvInt("HTTPAAS_SSH_RETRY", 5),
		// Default wait: allow long VM boot times. We wait at least the initial delay,
		// and allow SSH wait timeout to cover longer boots.
		SSHWaitTimeoutSeconds:   getenvInt("HTTPAAS_SSH_WAIT", 300),
		InitialBootDelaySeconds: getenvInt("HTTPAAS_INITIAL_BOOT_DELAY", 220),
	}
}

func getenv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getenvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvAny(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}
