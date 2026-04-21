# pdu-agent

Tiny Go daemon that listens for the proprietary shutdown broadcast emitted by an **ATEN PE8208G** (and likely the rest of the eco-PDU PE series) when an outlet configured for Wake-on-LAN mode is told to turn off via SNMP or the web UI.

The PDU's normal "Safe Shutdown" feature relies on ATEN's own binary agent running on every host. This is a clean-room replacement.

Companion to the [pdu-controller](https://github.com/kolobus/aten-pe8208g-webui) web app, but works on its own — you can keep using ATEN's native UI to trigger shutdowns and just replace the host-side agent.

## Protocol

Reverse-engineered from packet captures (see `docs/` in the companion repo).

**Shutdown signal (what we listen for):**

```
Transport:  UDP broadcast (255.255.255.255:9000)
Source:     the PDU's IP
Payload:    19 bytes
    00   01 00                 header / version
    02   44 53 70 80            magic  "DSp\x80"
    06   01 00                 flags
    08   07                    command  (0x07 = shutdown)
    09   00 00 00              padding
    12   06                    MAC length
    13   <6 bytes>             target MAC
```

**Wake signal**: standard Wake-on-LAN magic packet on UDP broadcast port 9. No agent needed — every WOL-capable NIC handles it natively. Not implemented here.

## Security caveat

ATEN's design assumes a trusted LAN: the shutdown packet is **plaintext broadcast with no auth or nonce**. Anyone who sniffs one packet can replay it and shut down any host running an agent with that MAC. This daemon mitigates somewhat by:

- Filtering on the PDU's source IP (`pdu_ip` in config).
- Pinning the listen socket to a specific NIC (`SO_BINDTODEVICE`, Linux only — on macOS the socket listens on all interfaces and only the source-IP filter gates traffic).

Best practice: keep management / PDU traffic on a dedicated VLAN.

## Configuration

Single file at `/etc/pdu-agent.conf` (path overridable with `-c`):

```ini
# required
nic          = eth0                     # NIC to bind to; local MAC for match
                                        # (typically "en0" on macOS)
pdu_ip       = 10.42.2.28               # only accept packets from this source IP
shutdown_cmd = /sbin/shutdown -h now    # runs via `sh -c` when triggered
```

Every trigger is logged with source IP and target MAC.

## Install

Prebuilt packages are published on every tagged release via goreleaser. Pick the format matching your distro.

### Debian / Ubuntu

```bash
wget https://github.com/kolobus/pdu-shutdown-agent/releases/latest/download/pdu-agent_<VERSION>_linux_amd64.deb
sudo apt install ./pdu-agent_<VERSION>_linux_amd64.deb
sudo cp /etc/pdu-agent.conf.example /etc/pdu-agent.conf
sudo $EDITOR /etc/pdu-agent.conf
sudo systemctl enable --now pdu-agent
```

### RHEL / Rocky / Fedora

```bash
sudo rpm -i https://github.com/kolobus/pdu-shutdown-agent/releases/latest/download/pdu-agent_<VERSION>_linux_amd64.rpm
sudo cp /etc/pdu-agent.conf.example /etc/pdu-agent.conf
sudo $EDITOR /etc/pdu-agent.conf
sudo systemctl enable --now pdu-agent
```

### Alpine

```bash
wget https://github.com/kolobus/pdu-shutdown-agent/releases/latest/download/pdu-agent_<VERSION>_linux_amd64.apk
sudo apk add --allow-untrusted ./pdu-agent_<VERSION>_linux_amd64.apk
```

### Raw binary (any Linux)

```bash
curl -L https://github.com/kolobus/pdu-shutdown-agent/releases/latest/download/pdu-agent_<VERSION>_linux_amd64.tar.gz \
  | sudo tar -xzC /usr/bin pdu-agent
sudo curl -o /etc/pdu-agent.conf https://raw.githubusercontent.com/kolobus/pdu-shutdown-agent/main/pdu-agent.conf.example
sudo curl -o /etc/systemd/system/pdu-agent.service https://raw.githubusercontent.com/kolobus/pdu-shutdown-agent/main/systemd/pdu-agent.service
sudo systemctl daemon-reload && sudo systemctl enable --now pdu-agent
```

arm64 and armv7 variants are also published (`…_linux_arm64`, `…_linux_armv7`).

### macOS (Homebrew)

```bash
brew install kolobus/tap/pdu-agent
sudo cp "$(brew --prefix)/etc/pdu-agent.conf.example" "$(brew --prefix)/etc/pdu-agent.conf"
sudo $EDITOR "$(brew --prefix)/etc/pdu-agent.conf"    # nic is usually en0
sudo brew services start pdu-agent
```

Runs as a system `LaunchDaemon` (needs root to invoke `/sbin/shutdown`). Logs land in `$(brew --prefix)/var/log/pdu-agent.log`.

### macOS (raw binary)

A universal (arm64 + amd64) build is published as `pdu-agent_<VERSION>_darwin_all.tar.gz`:

```bash
curl -L https://github.com/kolobus/pdu-shutdown-agent/releases/latest/download/pdu-agent_<VERSION>_darwin_all.tar.gz | tar -xz
sudo xattr -d com.apple.quarantine pdu-agent                             # clear Gatekeeper
sudo install -m 0755 pdu-agent /usr/local/bin/pdu-agent
sudo cp pdu-agent.conf.example /etc/pdu-agent.conf
sudo $EDITOR /etc/pdu-agent.conf
sudo cp launchd/net.fedorov.pdu-agent.plist /Library/LaunchDaemons/
sudo launchctl load -w /Library/LaunchDaemons/net.fedorov.pdu-agent.plist
```

## Build from source

```bash
go build -o pdu-agent
# Or cross-compile manually:
GOOS=linux GOARCH=amd64 go build -o dist/pdu-agent-linux-amd64
GOOS=linux GOARCH=arm64 go build -o dist/pdu-agent-linux-arm64
GOOS=darwin GOARCH=arm64 go build -o dist/pdu-agent-darwin-arm64
# Or run the full goreleaser pipeline locally (requires goreleaser):
goreleaser release --snapshot --clean
```

Static binary, no runtime deps.

## Testing without an actual PDU

```bash
# in one shell
sudo ./pdu-agent -c ./pdu-agent.conf.test

# in another — spoof a shutdown as if from the PDU
python3 -c "
import socket
mac = bytes.fromhex('bc2411ff2896')        # your NIC MAC
pkt = b'\\x01\\x00DSp\\x80\\x01\\x00\\x07\\x00\\x00\\x00\\x06' + mac
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
s.sendto(pkt, ('255.255.255.255', 9000))
"
```

(For a real unit test, override `shutdown_cmd` with `logger -t pdu-agent-test` and watch syslog.)

## License

MIT — see [LICENSE](LICENSE).

---

> Hi ATEN 👋 — if this repo surfaced in your monitoring, consider it friendly. The protocol RE here is harmless hobby work for homelab folks whose warranty long since expired. If you want to turn this into something official, or send over an **EA1140** / **EA1240** / **EA1640** sensor so I can document the RJ-11 side too, I'm at the email in the LICENSE.
