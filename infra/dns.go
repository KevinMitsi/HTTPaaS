package infra

import "fmt"

func AddDNS(sshClient *SSHClient, dnsIP, domain, host, ip string) error {
	cmd := fmt.Sprintf(
		"printf 'server 127.0.0.1\nzone %s.\nupdate add %s.%s. 3600 A %s\nsend\n' | sudo nsupdate -k /etc/bind/rndc.key",
		domain,
		host,
		domain,
		ip,
	)

	_, err := sshClient.Run(dnsIP, cmd)
	return err
}

func DeleteDNS(sshClient *SSHClient, dnsIP, domain, host string) error {
	cmd := fmt.Sprintf(
		"printf 'server 127.0.0.1\nzone %s.\nupdate delete %s.%s. A\nsend\n' | sudo nsupdate -k /etc/bind/rndc.key",
		domain,
		host,
		domain,
	)

	_, err := sshClient.Run(dnsIP, cmd)
	return err
}
