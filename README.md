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
- Pinning the listen socket to a specific NIC (`SO_BINDTODEVICE`).

Best practice: keep management / PDU traffic on a dedicated VLAN.

## Configuration

Single file at `/etc/pdu-agent.conf` (path overridable with `-c`):

```ini
# required
nic          = eth0                     # NIC to bind to; local MAC for match
pdu_ip       = 10.42.2.28               # only accept packets from this source IP
shutdown_cmd = /sbin/shutdown -h now    # runs via `sh -c` when triggered
```

Every trigger is logged with source IP and target MAC.

## Build

```bash
go build -o pdu-agent
# Or pinned architectures for your rack:
GOOS=linux GOARCH=amd64 go build -o dist/pdu-agent-linux-amd64
GOOS=linux GOARCH=arm64 go build -o dist/pdu-agent-linux-arm64
```

Static binary, no runtime deps.

## Install (systemd)

```bash
sudo install -m 755 pdu-agent /usr/local/bin/pdu-agent
sudo install -m 644 pdu-agent.conf.example /etc/pdu-agent.conf
sudo $EDITOR /etc/pdu-agent.conf
sudo install -m 644 systemd/pdu-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now pdu-agent
journalctl -u pdu-agent -f
```

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
