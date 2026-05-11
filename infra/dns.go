package infra

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	dnsSOABlockRegex   = regexp.MustCompile(`(?ms)SOA\s+ns1\.cloud\.local\.\s+admin\.cloud\.local\.\s*\(\s*(.*?)\s*\)`)
	dnsSerialLineRegex = regexp.MustCompile(`(?m)^\s+(\d+)\s*$`)
	dnsHostRecordRegex = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9.-]+)\s+IN\s+A\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s*$`)
)

func AddDNS(sshClient *SSHClient, dnsIP, domain, host, ip string) error {
	zoneFile := fmt.Sprintf("/etc/bind/db.%s", domain)
	state, err := loadDNSState(sshClient, dnsIP, zoneFile)
	if err != nil {
		return err
	}

	state.HostRecords[host] = ip
	state.Serial++

	return writeDNSState(sshClient, dnsIP, domain, zoneFile, state)
}

func DeleteDNS(sshClient *SSHClient, dnsIP, domain, host string) error {
	zoneFile := fmt.Sprintf("/etc/bind/db.%s", domain)
	state, err := loadDNSState(sshClient, dnsIP, zoneFile)
	if err != nil {
		return err
	}

	delete(state.HostRecords, host)
	state.Serial++

	return writeDNSState(sshClient, dnsIP, domain, zoneFile, state)
}

type dnsState struct {
	Serial      int
	HostRecords map[string]string
}

func loadDNSState(sshClient *SSHClient, dnsIP, zoneFile string) (dnsState, error) {
	output, err := sshClient.Run(dnsIP, fmt.Sprintf("cat %s", shellQuote(zoneFile)))
	if err != nil {
		if strings.Contains(output, "No such file") || strings.Contains(output, "cannot open") {
			return dnsState{
				Serial:      1,
				HostRecords: map[string]string{},
			}, nil
		}
		return dnsState{}, fmt.Errorf("leer zona DNS %s: %w: %s", zoneFile, err, strings.TrimSpace(output))
	}

	state, parseErr := parseDNSState(output)
	if parseErr != nil {
		return dnsState{}, fmt.Errorf("parsear zona DNS %s: %w", zoneFile, parseErr)
	}

	return state, nil
}

func parseDNSState(content string) (dnsState, error) {
	soaBlockMatch := dnsSOABlockRegex.FindStringSubmatch(content)
	if len(soaBlockMatch) != 2 {
		return dnsState{}, fmt.Errorf("no se pudo ubicar el bloque SOA")
	}

	serialMatch := dnsSerialLineRegex.FindStringSubmatch(soaBlockMatch[1])
	if len(serialMatch) != 2 {
		return dnsState{}, fmt.Errorf("no se pudo leer el serial actual")
	}

	serial, err := strconv.Atoi(serialMatch[1])
	if err != nil {
		return dnsState{}, fmt.Errorf("serial invalido: %w", err)
	}

	records := map[string]string{}
	for _, match := range dnsHostRecordRegex.FindAllStringSubmatch(content, -1) {
		if len(match) != 3 {
			continue
		}
		host := strings.TrimSpace(match[1])
		address := strings.TrimSpace(match[2])
		if host == "ns1.cloud.local." {
			continue
		}
		records[host] = address
	}

	return dnsState{
		Serial:      serial,
		HostRecords: records,
	}, nil
}

func writeDNSState(sshClient *SSHClient, dnsIP, domain, zoneFile string, state dnsState) error {
	content := buildDNSZoneContent(state.Serial, state.HostRecords)
	if err := sshClient.SCPBytes(dnsIP, []byte(content), "/tmp/db.cloud.local.tmp"); err != nil {
		return fmt.Errorf("copiar zona temporal: %w", err)
	}

	cmd := "mv /tmp/db.cloud.local.tmp /etc/bind/db.cloud.local && named-checkzone cloud.local /etc/bind/db.cloud.local && rndc reload cloud.local"

	output, err := sshClient.Run(dnsIP, cmd)
	if err != nil {
		return fmt.Errorf("actualizar zona DNS %s: %w: %s", zoneFile, err, strings.TrimSpace(output))
	}

	return nil
}

func buildDNSZoneContent(serial int, hostRecords map[string]string) string {
	var builder strings.Builder
	builder.WriteString("$TTL 604800\n")
	builder.WriteString("@       IN      SOA     ns1.cloud.local. admin.cloud.local. (\n")
	fmt.Fprintf(&builder, "                        %d\n", serial)
	builder.WriteString("                        604800\n")
	builder.WriteString("                        86400\n")
	builder.WriteString("                        2419200\n")
	builder.WriteString("                        604800 )\n")
	builder.WriteString("@               IN      NS      ns1.cloud.local.\n")
	builder.WriteString("ns1.cloud.local.        IN      A       192.168.10.10\n")

	keys := make([]string, 0, len(hostRecords))
	for host := range hostRecords {
		keys = append(keys, host)
	}
	sort.Strings(keys)

	for _, host := range keys {
		fmt.Fprintf(&builder, "%s        IN      A       %s\n", host, hostRecords[host])
	}

	return builder.String()
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
