package infra

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type VBox struct {
	Path string
}

func NewVBox(path string) *VBox {
	return &VBox{Path: path}
}

func (v *VBox) EnsureSnapshot(templateName, snapshotName string) error {
	exists, err := v.snapshotExists(templateName, snapshotName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return v.run("snapshot", templateName, "take", snapshotName, "--live")
}

func (v *VBox) CloneLinkedVM(templateName, snapshotName, newName string) error {
	if err := v.EnsureSnapshot(templateName, snapshotName); err != nil {
		return err
	}
	return v.run("clonevm", templateName, "--snapshot", snapshotName, "--options", "link", "--name", newName, "--register")
}

func (v *VBox) ModifyVM(name string, args ...string) error {
	cmd := append([]string{"modifyvm", name}, args...)
	return v.run(cmd...)
}

func (v *VBox) StartVM(name string) error {
	return v.run("startvm", name, "--type", "headless")
}

func (v *VBox) WaitForGuestAdditions(name string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		version, err := v.GuestPropertyGet(name, "/VirtualBox/GuestAdd/Version")
		if err == nil && version != "" {
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout esperando Guest Additions en %s", name)
}

func (v *VBox) GuestPropertyGet(name, key string) (string, error) {
	if v.Path == "" {
		return "", fmt.Errorf("ruta VBoxManage no configurada")
	}

	cmd := exec.Command(v.Path, "guestproperty", "get", name, key)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("VBoxManage guestproperty get %s %s: %w: %s", name, key, err, strings.TrimSpace(string(output)))
	}

	text := strings.TrimSpace(string(output))
	if strings.Contains(text, "No value set!") {
		return "", nil
	}

	const prefix = "Value:"
	if strings.HasPrefix(text, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(text, prefix)), nil
	}

	return text, nil
}

func (v *VBox) ConfigureStaticIPViaGuestControl(name, username, password, iface, ip, netmask string) error {
	script := fmt.Sprintf(
		"printf 'auto lo\\niface lo inet loopback\\n\\nauto %s\\niface %s inet static\\n    address %s\\n    netmask %s\\n' > /etc/network/interfaces && systemctl restart networking",
		iface,
		iface,
		ip,
		netmask,
	)

	return v.GuestControlRunBash(name, username, password, script)
}

func (v *VBox) ConfigureStaticIPViaGuestControlWithRetry(name, username, password, iface, ip, netmask string, retries int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < retries; i++ {
		err := v.ConfigureStaticIPViaGuestControl(name, username, password, iface, ip, netmask)
		if err == nil {
			if i > 0 {
				log.Printf("guestcontrol configure IP %s succeeded on retry %d", name, i+1)
			}
			return nil
		}
		lastErr = err
		if i < retries-1 {
			log.Printf("guestcontrol configure IP %s retry %d/%d (esperando %s): %v", name, i+1, retries, interval, err)
			time.Sleep(interval)
		}
	}

	return fmt.Errorf("configuracion IP via guestcontrol fallo tras %d intentos: %w", retries, lastErr)
}

func (v *VBox) WaitForGuestControlReady(name, username, password string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := v.GuestControlRunBash(name, username, password, "true")
		if err == nil {
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout esperando guestcontrol en %s", name)
}

func (v *VBox) GuestControlRunBash(name, username, password, bashCmd string) error {
	if v.Path == "" {
		return fmt.Errorf("ruta VBoxManage no configurada")
	}

	args := []string{
		"guestcontrol", name, "run",
		"--exe", "/bin/bash",
		"--username", username,
		"--password", password,
		"--putenv", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"--", "-c", bashCmd,
	}

	cmd := exec.Command(v.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("VBoxManage %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (v *VBox) MachineState(name string) (string, error) {
	if v.Path == "" {
		return "", fmt.Errorf("ruta VBoxManage no configurada")
	}

	cmd := exec.Command(v.Path, "showvminfo", name, "--machinereadable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("VBoxManage showvminfo %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "VMState=") {
			state := strings.TrimPrefix(line, "VMState=")
			state = strings.TrimSpace(state)
			state = strings.Trim(state, `"`)
			return strings.TrimSpace(state), nil
		}
	}

	return "", fmt.Errorf("no se pudo leer el estado de %s", name)
}

func (v *VBox) WaitUntilUnlocked(name string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := v.MachineState(name)
		if err != nil {
			return err
		}
		if state == "poweroff" {
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout esperando que %s quede apagada", name)
}

func (v *VBox) PowerOff(name string) error {
	return v.run("controlvm", name, "poweroff")
}

func (v *VBox) UnregisterDelete(name string) error {
	return v.run("unregistervm", name, "--delete")
}

// UnregisterDeleteRetry will attempt to unregister the VM several times,
// sleeping between attempts. Useful when VBoxManage reports the machine is locked.
func (v *VBox) UnregisterDeleteRetry(name string, retries int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < retries; i++ {
		err := v.UnregisterDelete(name)
		if err == nil {
			if i > 0 {
				log.Printf("unregistervm %s succeeded on retry %d", name, i+1)
			}
			return nil
		}
		lastErr = err
		if i < retries-1 {
			log.Printf("unregistervm %s retry %d/%d (esperando %s): %v", name, i+1, retries, interval, err)
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("unregistervm failed after %d retries: %w", retries, lastErr)
}

func (v *VBox) run(args ...string) error {
	if v.Path == "" {
		return fmt.Errorf("ruta VBoxManage no configurada")
	}

	cmd := exec.Command(v.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("VBoxManage %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (v *VBox) snapshotExists(templateName, snapshotName string) (bool, error) {
	if v.Path == "" {
		return false, fmt.Errorf("ruta VBoxManage no configurada")
	}

	cmd := exec.Command(v.Path, "snapshot", templateName, "list", "--machinereadable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("VBoxManage snapshot %s list: %w: %s", templateName, err, strings.TrimSpace(string(output)))
	}

	text := string(output)
	return strings.Contains(text, fmt.Sprintf("SnapshotName=\"%s\"", snapshotName)) || strings.Contains(text, fmt.Sprintf("Name=\"%s\"", snapshotName)), nil
}

func (v *VBox) EnsureDNSServerRunning(dnsServerName string) error {
	state, err := v.MachineState(dnsServerName)
	if err != nil {
		return fmt.Errorf("no se pudo obtener estado de %s: %w", dnsServerName, err)
	}

	if state == "running" {
		log.Printf("[STARTUP] %s ya está encendida", dnsServerName)
		return nil
	}

	log.Printf("[STARTUP] %s no está encendida (estado: %s), iniciando...", dnsServerName, state)
	if err := v.StartVM(dnsServerName); err != nil {
		return fmt.Errorf("no se pudo encender %s: %w", dnsServerName, err)
	}

	log.Printf("[STARTUP] %s iniciada correctamente", dnsServerName)
	return nil
}
