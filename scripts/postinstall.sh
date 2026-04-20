#!/bin/sh
set -e

# Only reload/enable when systemd is actually present.
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

if [ ! -f /etc/pdu-agent.conf ]; then
    cat <<EOF >&2

pdu-agent installed.

Next steps:
    1. sudo cp /etc/pdu-agent.conf.example /etc/pdu-agent.conf
    2. Edit /etc/pdu-agent.conf (nic, pdu_ip, shutdown_cmd).
    3. sudo systemctl enable --now pdu-agent
    4. journalctl -u pdu-agent -f

EOF
fi

exit 0
