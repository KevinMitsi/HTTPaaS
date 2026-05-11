package infra

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ConfigureHostname(sshClient *SSHClient, hostIP, hostname string) error {
	cmd := fmt.Sprintf("sudo hostnamectl set-hostname %s && echo '127.0.1.1 %s' | sudo tee -a /etc/hosts", hostname, hostname)
	_, err := sshClient.Run(hostIP, cmd)
	return err
}

func DeployContent(sshClient *SSHClient, hostIP, hostname, localZipPath string) error {
	tmpDir, err := os.MkdirTemp("", "httpaas-extract-*")
	if err != nil {
		return fmt.Errorf("crear directorio temporal: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZipToDir(localZipPath, tmpDir); err != nil {
		return err
	}

	contentRoot, err := flattenIfSingleRoot(tmpDir)
	if err != nil {
		return fmt.Errorf("detectar raiz del contenido: %w", err)
	}

	remoteDeployDir := "/tmp/deploy"
	if _, err := sshClient.Run(hostIP, fmt.Sprintf("rm -rf %q && mkdir -p %q", remoteDeployDir, remoteDeployDir)); err != nil {
		return err
	}

	if err := sshClient.CopyDir(hostIP, contentRoot, remoteDeployDir); err != nil {
		return err
	}

	cmd := `sh -lc 'rm -rf /var/www/html/* && cp -r /tmp/deploy/. /var/www/html/ && chown -R www-data:www-data /var/www/html/ && rm -rf /tmp/deploy && systemctl restart apache2'`
	_, err = sshClient.Run(hostIP, cmd)
	return err
}

func flattenIfSingleRoot(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir, err
	}

	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(dir, entries[0].Name()), nil
	}

	return dir, nil
}

func extractZipToDir(zipPath, dstDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("abrir zip: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath, err := safeZipPath(dstDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("crear directorio extraido: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("crear directorio padre: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("abrir archivo zip: %w", err)
		}

		out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("crear archivo extraido: %w", err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("copiar contenido zip: %w", err)
		}
		out.Close()
		rc.Close()
	}

	return nil
}

func safeZipPath(dstDir, name string) (string, error) {
	cleanName := filepath.Clean(name)
	if cleanName == "." {
		return dstDir, nil
	}

	targetPath := filepath.Join(dstDir, cleanName)
	relPath, err := filepath.Rel(dstDir, targetPath)
	if err != nil {
		return "", fmt.Errorf("validar ruta extraida: %w", err)
	}
	if relPath == ".." || len(relPath) >= 3 && relPath[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("zip inseguro: %s", name)
	}

	return targetPath, nil
}

func ConfigureVirtualHost(sshClient *SSHClient, hostIP, hostname, domain string) error {
	vhost := fmt.Sprintf(`<VirtualHost *:80>
    ServerName %s.%s
    DocumentRoot /var/www/html/%s
    <Directory /var/www/html/%s>
        AllowOverride All
        Require all granted
    </Directory>
</VirtualHost>`, hostname, domain, hostname, hostname)

	cmd := fmt.Sprintf("sudo sh -c 'cat > /etc/apache2/sites-available/%s.conf << \"EOF\"\n%s\nEOF\n' && sudo a2ensite %s.conf && sudo systemctl reload apache2", hostname, vhost, hostname)
	_, err := sshClient.Run(hostIP, cmd)
	return err
}

func ConfigureStaticNetworking(sshClient *SSHClient, hostIP, hostname, ip, dnsServer string) error {
	cmd := fmt.Sprintf(
		`sudo sh -lc 'cat > /etc/network/interfaces <<"EOF"
source /etc/network/interfaces.d/*

auto lo
iface lo inet loopback

auto enp0s3
iface enp0s3 inet static
    address %s
    netmask 255.255.255.0
    gateway 192.168.10.1
    dns-nameservers %s
EOF

printf "nameserver %s\n" > /etc/resolv.conf
echo "%s" > /etc/hostname
sed -i "/^127.0.1.1/d" /etc/hosts
printf "127.0.1.1 %s\n" >> /etc/hosts
nohup systemctl restart networking >/tmp/networking-restart.log 2>&1 &
'`,
		ip,
		dnsServer,
		dnsServer,
		hostname,
		hostname,
	)
	_, err := sshClient.Run(hostIP, cmd)
	return err
}
