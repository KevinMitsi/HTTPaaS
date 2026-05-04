package infra

import (
	"fmt"
	"os/exec"
	"strings"
)

type VBox struct {
	Path string
}

func NewVBox(path string) *VBox {
	return &VBox{Path: path}
}

func (v *VBox) CloneVM(templateName, newName string) error {
	return v.run("clonevm", templateName, "--name", newName, "--register")
}

func (v *VBox) ModifyVM(name string, args ...string) error {
	cmd := append([]string{"modifyvm", name}, args...)
	return v.run(cmd...)
}

func (v *VBox) StartVM(name string) error {
	return v.run("startvm", name, "--type", "headless")
}

func (v *VBox) PowerOff(name string) error {
	return v.run("controlvm", name, "poweroff")
}

func (v *VBox) UnregisterDelete(name string) error {
	return v.run("unregistervm", name, "--delete")
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
