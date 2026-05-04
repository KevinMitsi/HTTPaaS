package infra

import "fmt"

func ConfigureHostname(sshClient *SSHClient, hostIP, hostname string) error {
	cmd := fmt.Sprintf("sudo hostnamectl set-hostname %s && echo '127.0.1.1 %s' | sudo tee -a /etc/hosts", hostname, hostname)
	_, err := sshClient.Run(hostIP, cmd)
	return err
}

func DeployContent(sshClient *SSHClient, hostIP, hostname, remoteZip string) error {
	cmd := fmt.Sprintf(
		"sudo mkdir -p /var/www/html/%s && sudo unzip -o %s -d /var/www/html/%s/ && sudo chown -R www-data:www-data /var/www/html/%s/",
		hostname,
		remoteZip,
		hostname,
		hostname,
	)
	_, err := sshClient.Run(hostIP, cmd)
	return err
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
