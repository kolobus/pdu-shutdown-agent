// pdu-agent — listens for the proprietary ATEN PE8208G shutdown broadcast
// (UDP :9000, 19-byte payload with magic "DSp\x80" + command 0x07 + target MAC),
// matches the target MAC against the local NIC's hardware address, and runs a
// configured shell command (typically `shutdown -h now`) when the trigger fires.
//
// See README.md for the protocol breakdown and /etc/pdu-agent.conf for config.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

const (
	defaultConfigPath = "/etc/pdu-agent.conf"
	listenPort        = 9000
	payloadLen        = 19
	magicOffset       = 2
	commandOffset     = 8
	macLenOffset      = 12
	macOffset         = 13
	commandShutdown   = 0x07
)

var magic = []byte{0x44, 0x53, 0x70, 0x80} // "DSp" + 0x80

type config struct {
	NIC         string
	PDU         net.IP
	ShutdownCmd string
}

func main() {
	configPath := flag.String("c", defaultConfigPath, "path to config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	iface, err := net.InterfaceByName(cfg.NIC)
	if err != nil {
		log.Fatalf("interface %q: %v", cfg.NIC, err)
	}
	mac := iface.HardwareAddr
	if len(mac) != 6 {
		log.Fatalf("interface %q has no usable MAC address (%v)", cfg.NIC, mac)
	}

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: listenPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("listen :%d: %v", listenPort, err)
	}
	defer conn.Close()

	if err := bindToDevice(conn, cfg.NIC); err != nil {
		log.Printf("warn: SO_BINDTODEVICE on %s failed (%v); listening on all interfaces",
			cfg.NIC, err)
	}

	log.Printf("pdu-agent ready: nic=%s mac=%s pdu=%s cmd=%q",
		cfg.NIC, mac, cfg.PDU, cfg.ShutdownCmd)

	buf := make([]byte, 1500)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read: %v", err)
			continue
		}
		handle(buf[:n], src, mac, cfg)
	}
}

func handle(payload []byte, src *net.UDPAddr, localMAC net.HardwareAddr, cfg *config) {
	if !src.IP.Equal(cfg.PDU) {
		return // source IP filter: silently drop anything not from our PDU
	}
	if len(payload) < payloadLen {
		return
	}
	if !bytes.Equal(payload[magicOffset:magicOffset+len(magic)], magic) {
		return
	}
	if payload[commandOffset] != commandShutdown {
		log.Printf("ignored: PDU sent command 0x%02x (expected 0x07 shutdown) from %s",
			payload[commandOffset], src.IP)
		return
	}
	if payload[macLenOffset] != 6 {
		return
	}
	target := net.HardwareAddr(payload[macOffset : macOffset+6])
	if !bytes.Equal(target, localMAC) {
		return // someone else's MAC, not for us
	}

	log.Printf("SHUTDOWN trigger from %s (target=%s). Running: %s",
		src.IP, target, cfg.ShutdownCmd)
	if err := runShell(cfg.ShutdownCmd); err != nil {
		log.Printf("shutdown command failed: %v", err)
	}
}

func runShell(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func loadConfig(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &config{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
		switch key {
		case "nic":
			cfg.NIC = val
		case "pdu_ip":
			cfg.PDU = net.ParseIP(val)
			if cfg.PDU == nil {
				return nil, fmt.Errorf("invalid pdu_ip: %q", val)
			}
		case "shutdown_cmd":
			cfg.ShutdownCmd = val
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if cfg.NIC == "" {
		return nil, fmt.Errorf("nic is required")
	}
	if cfg.PDU == nil {
		return nil, fmt.Errorf("pdu_ip is required")
	}
	if cfg.ShutdownCmd == "" {
		return nil, fmt.Errorf("shutdown_cmd is required")
	}
	return cfg, nil
}
