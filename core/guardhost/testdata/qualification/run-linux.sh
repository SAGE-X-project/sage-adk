#!/bin/sh
# Separate-account qualification on a Linux host with passwordless sudo.
# Usage: run-linux.sh BIN_DIR LOG_FILE
# BIN_DIR holds adk-signer, adk-approve and qualification for this host.
# Accounts (uid:gid): signer-a 1001:2001, signer-b 1002:2002 (also holds
# the receiver's X25519 KEM key), receiver
# 1003:2002, caller 1004:2001, operator 1005:1005. Test-only: the Registry
# Source and calculator measurement are synthetic fixtures.
set -eu
BIN=$1
LOG=$2
: > "$LOG"
log() { echo "### $*" | tee -a "$LOG"; }
as() { id=$1; shift; sudo setpriv --reuid="${id%%:*}" --regid="${id##*:}" --clear-groups "$@"; }

sudo rm -rf /srv/sq
sudo mkdir -p /srv/sq/bin /srv/sq/signer-a /srv/sq/signer-b /srv/sq/sign-a /srv/sq/sign-b /srv/sq/operator /srv/sq/shared \
  /srv/sq/receiver/state /srv/sq/receiver/artifacts /srv/sq/caller/state /srv/sq/caller/artifacts
sudo cp "$BIN"/adk-signer "$BIN"/adk-approve "$BIN"/qualification /srv/sq/bin/
sudo chmod 0755 /srv/sq /srv/sq/bin /srv/sq/bin/*
sudo chown 1001:1001 /srv/sq/signer-a && sudo chmod 0700 /srv/sq/signer-a
sudo chown 1002:1002 /srv/sq/signer-b && sudo chmod 0700 /srv/sq/signer-b
sudo chown 1001:2001 /srv/sq/sign-a && sudo chmod 0750 /srv/sq/sign-a
sudo chown 1002:2002 /srv/sq/sign-b && sudo chmod 0750 /srv/sq/sign-b
sudo chown 1005:1005 /srv/sq/operator && sudo chmod 0700 /srv/sq/operator
sudo chown -R 1003:2002 /srv/sq/receiver && sudo chmod 0700 /srv/sq/receiver /srv/sq/receiver/state /srv/sq/receiver/artifacts
sudo chown -R 1004:2001 /srv/sq/caller && sudo chmod 0700 /srv/sq/caller /srv/sq/caller/state /srv/sq/caller/artifacts
Q=/srv/sq/bin/qualification
log "host $(uname -srm)"

log "clock observation for the whole run"
"$Q" clockwatch -for 480s > /tmp/sq-clock.out 2>&1 &
CLOCK=$!

A=$(as 1001:1001 /srv/sq/bin/adk-signer keygen -key /srv/sq/signer-a/seed)
B=$(as 1002:1002 /srv/sq/bin/adk-signer keygen -key /srv/sq/signer-b/seed)
OP=$(as 1005:1005 /srv/sq/bin/adk-approve keygen -key /srv/sq/operator/seed)
KEM=$(as 1002:1002 /srv/sq/bin/adk-signer kemgen -key /srv/sq/signer-b/kem)
log "alice=$A bob=$B operator=$OP kem=$KEM"

sudo "$Q" setup -dir /srv/sq/shared -alice "$A" -bob "$B" -kem "$KEM" | tee -a "$LOG"
sudo chmod 0777 /srv/sq/shared
as 1005:1005 /srv/sq/bin/adk-approve sign -key /srv/sq/operator/seed -policy /srv/sq/shared/policy.json -manifest /srv/sq/shared/manifest.json -sequence 1 -out /srv/sq/shared/approval.json | tee -a "$LOG"
sudo chmod 0755 /srv/sq/shared
for h in receiver:1003:2002 caller:1004:2001; do
  name=${h%%:*}; owner=${h#*:}
  # Expand globs as root: the host artifact directories are mode 0700.
  sudo sh -c "cp /srv/sq/shared/artifacts/* /srv/sq/$name/artifacts/ && chown $owner /srv/sq/$name/artifacts/* && chmod 0600 /srv/sq/$name/artifacts/*"
done
sudo tee /srv/sq/shared/receiver.json >/dev/null <<EOF
{"State":"/srv/sq/receiver/state","Artifacts":"/srv/sq/receiver/artifacts","Shared":"/srv/sq/shared","Signer":"/srv/sq/sign-b/s","SignerUID":1002,"Approver":"$OP","Address":"127.0.0.1:7443","Create":true,"Wait":"0s"}
EOF
sudo tee /srv/sq/shared/caller.json >/dev/null <<EOF
{"State":"/srv/sq/caller/state","Artifacts":"/srv/sq/caller/artifacts","Shared":"/srv/sq/shared","Signer":"/srv/sq/sign-a/s","SignerUID":1001,"Approver":"$OP","Address":"127.0.0.1:7443","Create":true,"Wait":"365s"}
EOF

log "signers"
as 1001:2001 /srv/sq/bin/adk-signer serve -key /srv/sq/signer-a/seed -socket /srv/sq/sign-a/s -allow-uid 1004 -roles intent,transport > /tmp/sq-signer-a.out 2>&1 &
as 1002:2002 /srv/sq/bin/adk-signer serve -key /srv/sq/signer-b/seed -kem-key /srv/sq/signer-b/kem -socket /srv/sq/sign-b/s -allow-uid 1003 -roles result,transport,kem > /tmp/sq-signer-b.out 2>&1 &
sleep 2
cat /tmp/sq-signer-a.out /tmp/sq-signer-b.out | tee -a "$LOG"

log "receiver"
as 1003:2002 "$Q" receiver -config /srv/sq/shared/receiver.json > /tmp/sq-receiver.out 2>&1 &
sleep 2
cat /tmp/sq-receiver.out | tee -a "$LOG"

log "isolation checks"
as 1004:2001 sh -c 'cat /srv/sq/signer-a/seed >/dev/null 2>&1 && echo FAIL caller-read-signer-key || echo caller-cannot-read-signer-key' | tee -a "$LOG"
as 1003:2002 sh -c 'ls /srv/sq/caller/state >/dev/null 2>&1 && echo FAIL receiver-read-caller-state || echo receiver-cannot-read-caller-state' | tee -a "$LOG"
as 1003:2002 sh -c 'ls /srv/sq/sign-a >/dev/null 2>&1 && echo FAIL receiver-reached-signer-a || echo receiver-cannot-reach-signer-a' | tee -a "$LOG"
as 1003:2002 sh -c 'cat /srv/sq/signer-b/kem >/dev/null 2>&1 && echo FAIL receiver-read-kem-key || echo receiver-cannot-read-kem-key' | tee -a "$LOG"

log "caller (waits for the 360 s replay quarantine)"
set +e
as 1004:2001 "$Q" caller -config /srv/sq/shared/caller.json > /tmp/sq-caller.out 2>&1
rc=$?
set -e
cat /tmp/sq-caller.out | tee -a "$LOG"
log "caller-exit=$rc"

log "receiver state before shutdown"
if sudo pkill -TERM -f "^$Q receiver"; then echo receiver-was-running | tee -a "$LOG"; else echo receiver-not-running | tee -a "$LOG"; fi
sleep 3
cat /tmp/sq-receiver.out | tee -a "$LOG"
sudo pkill -TERM -f "^/srv/sq/bin/adk-signer serve" || true
wait "$CLOCK" || true
cat /tmp/sq-clock.out | tee -a "$LOG"
exit "$rc"
