#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop pdu-agent >/dev/null 2>&1 || true
    systemctl disable pdu-agent >/dev/null 2>&1 || true
fi

exit 0
