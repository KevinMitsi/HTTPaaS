package config

import (
	"os"
	"strconv"
)

type Config struct {
	BaseIP                string
	StartOctet            int
	Domain                string
	SSHKeyPath            string
	SSHUser               string
	SSHPort               int
	DNSServerIP           string
	TemplateName          string
	VBoxManagePath        string
	Port                  string
	StorePath             string
	UploadTmpDir          string
	SSHTimeoutSeconds     int
	SSHRetrySeconds       int
	SSHWaitTimeoutSeconds int
}

func Load() Config {
	return Config{
		BaseIP:                getenv("HTTPAAS_BASE_IP", "192.168.10."),
		StartOctet:            getenvInt("HTTPAAS_START_OCTET", 30),
		Domain:                getenv("HTTPAAS_DOMAIN", "cloud.local"),
		SSHKeyPath:            getenv("HTTPAAS_SSH_KEY", ""),
		SSHUser:               getenv("HTTPAAS_SSH_USER", "debian"),
		SSHPort:               getenvInt("HTTPAAS_SSH_PORT", 22),
		DNSServerIP:           getenv("HTTPAAS_DNS_IP", "192.168.10.10"),
		TemplateName:          getenv("HTTPAAS_TEMPLATE", "ApacheServer"),
		VBoxManagePath:        getenv("HTTPAAS_VBOXMANAGE", "VBoxManage"),
		Port:                  getenv("HTTPAAS_PORT", ":8080"),
		StorePath:             getenv("HTTPAAS_STORE", "./instances.json"),
		UploadTmpDir:          getenv("HTTPAAS_UPLOAD_TMP", "./uploads"),
		SSHTimeoutSeconds:     getenvInt("HTTPAAS_SSH_TIMEOUT", 120),
		SSHRetrySeconds:       getenvInt("HTTPAAS_SSH_RETRY", 3),
		SSHWaitTimeoutSeconds: getenvInt("HTTPAAS_SSH_WAIT", 120),
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
