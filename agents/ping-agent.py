#!/usr/bin/env python3
"""Uptime-monitor push agent.

Reads a list of server status events from a JSON file and POSTs them to
ping-service's push endpoint (POST /api/v1/ping/events). The access token is
short-lived (15 min), so the agent refreshes it via the auth-service refresh
endpoint using the refresh token.

Setup (operator, out of band):
  1. Obtain a ping refresh token once: login -> POST /api/v1/auth/login
     (app token) -> mint POST /api/v1/auth/sessions/ping (ping access +
     refresh token). Persist only the refresh token.
  2. export PING_REFRESH_TOKEN. The agent mints/refreshes the access token
     itself on startup and before it expires, so no access token is needed.

Env vars:
  PING_REFRESH_TOKEN  (required) ping-scoped refresh token, used on first
                      start or when no token file exists
  PING_TOKEN          (optional) access token; if unset the agent refreshes
                      one from PING_REFRESH_TOKEN on startup
  PING_TOKEN_FILE     (optional) where the agent persists the rotated token
                      pair; defaults to /var/lib/uptime-agent/tokens.json.
                      Delete this file to force re-reading PING_REFRESH_TOKEN.
  PING_ENDPOINT       push URL, default http://localhost:8080/api/v1/ping/events
  PING_AUTH_BASE      auth base URL, default http://localhost:8080/api/v1/auth
  PING_EVENTS_FILE    events file, default /etc/uptime-agent/events.json
  PING_EMPTY_INTERVAL seconds to wait when the file is empty/invalid, default 30

Events file format (JSON array):
  [ {"id": 1, "status": "ON"}, {"id": 2, "status": "OFF"} ]
"""

import base64
import json
import os
import signal
import sys
import time
import urllib.error
import urllib.request

REFRESH_SKEW = 60  # refresh this many seconds before the access token expires


def env(name, default):
    return os.environ.get(name, default)


PING_TOKEN = env("PING_TOKEN", "")
PING_REFRESH_TOKEN = env("PING_REFRESH_TOKEN", "")
PING_ENDPOINT = env("PING_ENDPOINT", "http://localhost:8080/api/v1/ping/events")
PING_AUTH_BASE = env("PING_AUTH_BASE", "http://localhost:8080/api/v1/auth")
PING_EVENTS_FILE = env("PING_EVENTS_FILE", "/etc/uptime-agent/events.json")
EMPTY_INTERVAL = float(env("PING_EMPTY_INTERVAL", "30"))


def log(msg):
    ts = time.strftime("%Y-%m-%dT%H:%M:%S", time.localtime())
    print(f"[{ts}] {msg}", flush=True)


def decode_exp(token):
    """Return the `exp` claim (unix seconds) from a JWT without verifying it."""
    try:
        _, payload, _ = token.split(".")
        padding = "=" * (-len(payload) % 4)
        data = json.loads(base64.urlsafe_b64decode(payload + padding))
        return int(data.get("exp", 0))
    except (ValueError, json.JSONDecodeError, base64.binascii.Error):
        return 0


def http_post(url, body, token=None):
    data = json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()
    except urllib.error.URLError as e:
        return 0, str(e.reason)


def refresh_token():
    global PING_TOKEN, PING_REFRESH_TOKEN
    status, raw = http_post(
        PING_AUTH_BASE.rstrip("/") + "/refresh",
        {"refresh_token": PING_REFRESH_TOKEN},
    )
    if status != 200:
        log(f"FATAL: token refresh failed (status={status}): {raw}")
        sys.exit(1)
    try:
        resp = json.loads(raw)
        PING_TOKEN = resp["access_token"]
        PING_REFRESH_TOKEN = resp["refresh_token"]
    except (json.JSONDecodeError, KeyError):
        log(f"FATAL: token refresh returned no tokens: {raw}")
        sys.exit(1)
    log("token refreshed")


def load_events():
    try:
        with open(PING_EVENTS_FILE) as f:
            data = json.load(f)
    except FileNotFoundError:
        log(f"events file {PING_EVENTS_FILE} not found; create it as a JSON array of {{id,status}}")
        return None
    except (json.JSONDecodeError, OSError) as e:
        log(f"cannot read events file: {e}")
        return None
    if not isinstance(data, list):
        log("events file must be a JSON array of {id, status}")
        return None
    return data


def sleep_until(unix_ms):
    secs = unix_ms / 1000.0 - time.time()
    if secs > 0:
        time.sleep(secs)


def main():
    if not PING_REFRESH_TOKEN:
        log("FATAL: PING_REFRESH_TOKEN is required")
        sys.exit(1)

    stop = {"sig": False}

    def handle_signal(_signum, _frame):
        stop["sig"] = True

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    log(f"agent started; endpoint={PING_ENDPOINT} events={PING_EVENTS_FILE}")

    if not PING_TOKEN:
        refresh_token()

    while not stop["sig"]:
        if time.time() >= decode_exp(PING_TOKEN) - REFRESH_SKEW:
            refresh_token()

        events = load_events()
        if not events:
            time.sleep(EMPTY_INTERVAL)
            continue

        log("sending sample payload: " + json.dumps(events))
        status, raw = http_post(PING_ENDPOINT, events, PING_TOKEN)
        log(f"response status={status} body={raw}")

        if status in (200, 207):
            try:
                sleep_until(int(json.loads(raw)["next_time"]))
                continue
            except (ValueError, KeyError, json.JSONDecodeError):
                pass
            time.sleep(EMPTY_INTERVAL)
        elif status == 429:
            try:
                sleep_until(int(json.loads(raw)["next_time"]))
            except (ValueError, KeyError, json.JSONDecodeError):
                time.sleep(EMPTY_INTERVAL)
        elif status == 400:
            time.sleep(EMPTY_INTERVAL)
        elif status == 401:
            refresh_token()
        else:  # 403, 5xx, network errors
            time.sleep(60)


if __name__ == "__main__":
    main()
