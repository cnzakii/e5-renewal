#!/bin/sh
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "Usage: $0 IMAGE [BASE_PATH]" >&2
  exit 2
fi

container=
volume=
cleanup() {
  status=$?
  trap - 0 HUP INT TERM
  if [ -n "$container" ]; then
    docker rm -f "$container" >/dev/null || status=1
  fi
  if [ -n "$volume" ]; then
    docker volume rm "$volume" >/dev/null || status=1
  fi
  exit "$status"
}
trap cleanup 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

name="e5-renewal-smoke-$(python3 -c 'import uuid; print(uuid.uuid4().hex)')"
base_path=${2:-}
volume=$(docker volume create "$name-data")
container=$(docker create --pull=never --name "$name" --restart=no \
  --publish 127.0.0.1::8080 --mount "type=volume,src=$volume,dst=/data" \
  --env E5_INITIAL_PASSWORD=container-smoke-password \
  --env "E5_PUBLIC_URL=http://127.0.0.1:8080$base_path" "$1")
case $(docker inspect --format '{{.Config.User}}' "$container") in
  65532|65532:65532) ;;
  *) echo "Runtime image must default to UID 65532." >&2; exit 1 ;;
esac

check() {
  binding=$(docker port "$container" 8080/tcp)
  python3 -I - "http://127.0.0.1:${binding##*:}$base_path" "$1" "$base_path" <<'PY'
import http.client
import http.cookiejar
import json
import re
import sys
import time
import urllib.error
import urllib.request
import urllib.parse

base, phase, prefix = sys.argv[1:]
client = urllib.request.build_opener(
    urllib.request.ProxyHandler({}),
    urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()),
)


def request(path, data=None, csrf="", expected=200):
    headers = {"Origin": "http://127.0.0.1:8080", "X-CSRF-Token": csrf}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(
        base + path, data=None if data is None else json.dumps(data).encode(), headers=headers
    )
    try:
        with client.open(req, timeout=3) as response:
            assert response.status == expected, f"Unexpected status for {path}"
            body = response.read((2 << 20) + 1)
            assert len(body) <= 2 << 20, f"Oversized response for {path}"
            return body
    except urllib.error.HTTPError as error:
        error.close()
        if error.code != expected:
            raise
        return b""


deadline = time.monotonic() + 30
while True:
    try:
        assert json.loads(request("/healthz"))["status"] == "ok", "Health check failed"
        break
    except (urllib.error.URLError, http.client.HTTPException, OSError):
        if time.monotonic() >= deadline:
            raise SystemExit("Container did not become healthy within 30 seconds.") from None
        time.sleep(0.2)

html = request("/")
assert b"<html" in html.lower(), "Embedded frontend is missing"
base_element = re.search(rb'<base\b[^>]*\bhref="([^"]+)"', html)
assert base_element and base_element[1].decode() == prefix + "/", "Frontend base path is wrong"
asset = re.search(rb'<script\b[^>]*\bsrc="([^"?#]+)"', html)
assert asset, "Embedded frontend has no bundled script"
asset_path = urllib.parse.urljoin(base + "/", asset[1].decode()).removeprefix(base)
assert request(asset_path), "Embedded frontend asset is empty"
request("/api/accounts", expected=401)
assert json.loads(request("/api/login", {"password": "container-smoke-password"}))["authenticated"]
session = json.loads(request("/api/session"))
assert session["authenticated"] and session["csrf"], "Login did not create a session"
accounts = json.loads(request("/api/accounts"))["accounts"]
if phase == "create":
    assert accounts == [], "Disposable storage was not empty"
    account = json.loads(request("/api/accounts", {
        "name": "Container smoke account",
        "auth_type": "delegated",
        "client_id": "00000000-0000-4000-8000-000000000001",
        "tenant_id": "smoke.invalid",
        "client_secret": "synthetic-smoke-secret",
        "enabled": False,
        "min_minutes": 120,
        "max_minutes": 240,
    }, csrf=session["csrf"], expected=201))
else:
    assert len(accounts) == 1, "Restart did not preserve the account"
    account = accounts[0]
assert account["name"] == "Container smoke account" and account["auth_type"] == "delegated"
assert not account["enabled"] and account["has_client_secret"], "Disabled credentials were not preserved"
assert "client_secret" not in account and "refresh_token" not in account, "API exposed credentials"
if phase == "create":
    assert json.loads(request("/api/runs"))["runs"] == [], "Disabled account unexpectedly ran"
    request(f"/api/accounts/{account['id']}/run", {}, csrf=session["csrf"], expected=202)

deadline = time.monotonic() + 10
while True:
    runs = json.loads(request("/api/runs"))["runs"]
    if len(runs) == 1 and runs[0]["status"] != "running":
        break
    assert time.monotonic() < deadline, "Execution did not finish or history was lost"
    time.sleep(0.1)

assert runs[0]["status"] == "failed", "Missing authorization did not fail cleanly"
assert runs[0]["error_code"] == "authorization_missing_refresh_token"
assert runs[0]["request_count"] == 0, "A token failure was counted as a Graph call"
request("/api/logout", {}, csrf=session["csrf"])
request("/api/accounts", expected=401)
PY
}

stop() {
  docker stop --time 15 "$container" >/dev/null
  state=$(docker inspect --format '{{.State.ExitCode}} {{.State.OOMKilled}} {{.State.Running}} {{.RestartCount}}' "$container")
  if [ "$state" != '0 false false 0' ]; then
    echo "Container failed graceful shutdown (exit/OOM/running/restarts: $state)." >&2
    exit 1
  fi
}

docker start "$container" >/dev/null
check create
stop
docker start "$container" >/dev/null
check persisted
stop
echo "Container smoke passed: non-root startup, prefix '$base_path', login, execution errors, persistent history, and graceful restart."
