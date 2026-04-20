#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    # If the service is already enabled/running, restart it so the new
    # unit file and binary take effect. try-restart is a no-op when the
    # service isn't running — safe for first-time installs.
    systemctl try-restart pdu-agent >/dev/null 2>&1 || true
    # Clear a previous failed state left over from the old broken unit.
    systemctl reset-failed pdu-agent >/dev/null 2>&1 || true
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
