#!/usr/bin/env bash
# openframe-sync.sh
# Decide whether OpenFrame should be running right now based on a fixed
# daily window, and apply the desired state. Intended to be triggered by
# systemd timers and at boot (via timers with Persistent=true).
#
# Defaults: 06:00 start, 20:00 stop. Override with env vars:
#   OPENFRAME_START_HHMM=HH:MM
#   OPENFRAME_STOP_HHMM=HH:MM
#
# Notes:
# - If within the active window: start openframe.service (which powers on TV
#   and selects HDMI via ExecStartPre hooks in openframe.service).
# - If outside: stop openframe.service and defensively send CEC standby.

set -euo pipefail

START_HHMM=${OPENFRAME_START_HHMM:-06:00}
STOP_HHMM=${OPENFRAME_STOP_HHMM:-20:00}

parse_hhmm() {
  local hhmm=$1
  local hh=${hhmm%%:*}
  local mm=${hhmm##*:}
  # Use base-10 to avoid issues with leading zeros
  echo $((10#${hh} * 60 + 10#${mm}))
}

now_h=$(date +%H)
now_m=$(date +%M)
now_min=$((10#${now_h} * 60 + 10#${now_m}))

start_min=$(parse_hhmm "${START_HHMM}")
stop_min=$(parse_hhmm "${STOP_HHMM}")

in_window=false
if (( start_min < stop_min )); then
  # Normal window (e.g., 06:00–20:00)
  if (( now_min >= start_min && now_min < stop_min )); then
    in_window=true
  fi
else
  # Window crosses midnight (e.g., 20:00–06:00)
  if (( now_min >= start_min || now_min < stop_min )); then
    in_window=true
  fi
fi

log() { echo "[openframe-sync] $*"; }

if [[ "${in_window}" == "true" ]]; then
  log "Within active window (${START_HHMM}–${STOP_HHMM}); starting slideshow."
  /usr/bin/systemctl --user start openframe.service || true
else
  log "Outside active window (${START_HHMM}–${STOP_HHMM}); stopping slideshow and powering TV off."
  /usr/bin/systemctl --user stop openframe.service || true
  # Defensive CEC: ensure TV is off
  /bin/sh -c 'echo "standby 0" | cec-client -s -d 1' || true
fi

