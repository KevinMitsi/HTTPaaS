package infra

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	user    string
	port    int
	timeout time.Duration
	signer  ssh.Signer
}

func NewSSHClient(user, keyPath string, port int, timeout time.Duration) (*SSHClient, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("leer llave SSH: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsear llave SSH: %w", err)
	}

	return &SSHClient{
		user:    user,
		port:    port,
		timeout: timeout,
		signer:  signer,
	}, nil
}

func (c *SSHClient) Run(host, cmd string) (string, error) {
	client, err := c.dial(host)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("crear sesion SSH: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("comando SSH: %w", err)
	}

	return string(output), nil
}

func (c *SSHClient) CopyFile(host, localPath, remotePath string) error {
	client, err := c.dial(host)
	if err != nil {
		return err
	}
	defer client.Close()

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("abrir archivo local: %w", err)
	}
	defer file.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("crear sesion SSH: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin SSH: %w", err)
	}

	if err := session.Start(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("iniciar copia SSH: %w", err)
	}

	if _, err := io.Copy(stdin, file); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("copiar archivo: %w", err)
	}
	_ = stdin.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("esperar copia SSH: %w", err)
	}

	return nil
}

func (c *SSHClient) WaitForSSH(host string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client, err := c.dial(host)
		if err == nil {
			_ = client.Close()
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout esperando SSH en %s", host)
}

func (c *SSHClient) dial(host string) (*ssh.Client, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", c.port))
	config := &ssh.ClientConfig{
		User:            c.user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.timeout,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial SSH %s: %w", addr, err)
	}

	return client, nil
}
