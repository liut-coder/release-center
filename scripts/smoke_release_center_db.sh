#!/usr/bin/env bash
set -euo pipefail

GO_CMD="${GO:-go}"
PSQL_CMD="${PSQL:-psql}"
DB_NAME="${SMOKE_DB_NAME:-game_helper_codex}"
DB_SCHEMA="${SMOKE_DB_SCHEMA:-release_center_mvp_smoke}"
DB_HOST="${SMOKE_DB_HOST:-/var/run/postgresql}"
ADDR="${SMOKE_ADDR:-127.0.0.1:18083}"
BASE_URL="http://${ADDR}"
FILE_ROOT="${SMOKE_FILE_ROOT:-/tmp/release-center-db-smoke-files}"
WEB_DIST="${WEB_DIST:-web/dist}"
DATABASE_URL="${SMOKE_DATABASE_URL:-postgres://root@/${DB_NAME}?host=${DB_HOST}&search_path=${DB_SCHEMA}}"
RESOURCE_VERSION="${SMOKE_RESOURCE_VERSION:-20260602.1}"
APK_SHA256="${SMOKE_APK_SHA256:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"

SERVER_PID=""
WORK_DIR="$(mktemp -d)"

cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  rm -rf "${WORK_DIR}" "${FILE_ROOT}"
  if [[ "${KEEP_SMOKE_SCHEMA:-}" != "true" ]]; then
    "${PSQL_CMD}" -d "${DB_NAME}" -c "drop schema if exists ${DB_SCHEMA} cascade;" >/dev/null
  fi
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 2
  fi
}

wait_ready() {
  for _ in $(seq 1 50); do
    if curl -fsS "${BASE_URL}/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "release center did not become ready at ${BASE_URL}" >&2
  return 1
}

json_post() {
  local path="$1"
  local body="$2"
  curl -fsS "${BASE_URL}${path}" \
    -H 'Content-Type: application/json' \
    -d "${body}"
}

require_cmd curl
require_cmd jq
require_cmd "${PSQL_CMD}"

"${PSQL_CMD}" -v ON_ERROR_STOP=1 -d "${DB_NAME}" \
  -c "drop schema if exists ${DB_SCHEMA} cascade; create schema ${DB_SCHEMA} authorization root;" >/dev/null

DATABASE_URL="${DATABASE_URL}" "${GO_CMD}" run -buildvcs=false ./cmd/migrate >/tmp/release-center-smoke-migrate.log

rm -rf "${FILE_ROOT}"
mkdir -p "${FILE_ROOT}"
ADDR="${ADDR}" \
FILE_ROOT="${FILE_ROOT}" \
WEB_DIST="${WEB_DIST}" \
DATABASE_URL="${DATABASE_URL}" \
"${GO_CMD}" run -buildvcs=false ./cmd/server >/tmp/release-center-smoke-server.log 2>&1 &
SERVER_PID="$!"
wait_ready

curl -fsS "${BASE_URL}/" >/dev/null

json_post "/admin/api/apps" \
  '{"app_key":"game-helper-android","name":"游戏助手","platform":"android","package_name":"com.kingdomhelper.executor","description":"smoke app","enabled":true}' \
  | tee "${WORK_DIR}/app.json" >/dev/null

"${GO_CMD}" run -buildvcs=false ./cmd/releasectl artifact-upload \
  -base-url "${BASE_URL}" \
  -token smoke \
  -artifact-url 'https://example.com/game-helper-0.1.0.apk' \
  -file-name game-helper-0.1.0.apk \
  -size-bytes 123456 \
  -sha256 "${APK_SHA256}" \
  -git-ref main \
  -git-commit abcdef1 \
  -version-name 0.1.0 \
  -version-code 100 \
  -build-number 1 \
  -channel dev \
  -artifact-type apk \
  | tee "${WORK_DIR}/artifact.json" >/dev/null
BUILD_ID="$(jq -r '.job.id' "${WORK_DIR}/artifact.json")"

json_post "/admin/api/app-releases" \
  "{\"build_id\":\"${BUILD_ID}\",\"channel\":\"dev\",\"title\":\"Smoke 0.1.0\",\"summary\":\"smoke release\",\"release_notes_markdown\":\"## Smoke\\n\\n- db release\",\"update_level\":\"recommended\",\"rollout_percentage\":100}" \
  | tee "${WORK_DIR}/release.json" >/dev/null
RELEASE_ID="$(jq -r '.release.id' "${WORK_DIR}/release.json")"

curl -fsS -X POST "${BASE_URL}/admin/api/app-releases/${RELEASE_ID}/publish" \
  | tee "${WORK_DIR}/publish.json" >/dev/null

json_post "/api/v1/app/update-check" \
  '{"deviceId":"device-smoke","versionCode":1,"channel":"dev","buildType":"debug"}' \
  | tee "${WORK_DIR}/update-check.json" >/dev/null
jq -e '.has_update == true and .version_code == 100 and .release.downloadUrl != ""' "${WORK_DIR}/update-check.json" >/dev/null

mkdir -p "${WORK_DIR}/resources/templates-common"
printf '{"scene":"smoke","enabled":true}\n' > "${WORK_DIR}/resources/templates-common/metadata.json"
"${GO_CMD}" run -buildvcs=false ./cmd/releasectl resource-pack \
  -root "${WORK_DIR}/resources" \
  -out "${WORK_DIR}/bundle" \
  -version "${RESOURCE_VERSION}" \
  -channel dev \
  -title 'Smoke resources' \
  -summary 'resource smoke' \
  | tee "${WORK_DIR}/resource-pack.json" >/dev/null

"${GO_CMD}" run -buildvcs=false ./cmd/releasectl resource-upload \
  -base-url "${BASE_URL}" \
  -token smoke \
  -bundle "${WORK_DIR}/bundle/bundle-${RESOURCE_VERSION}.json" \
  -publish \
  | tee "${WORK_DIR}/resource-upload.json" >/dev/null
RESOURCE_ID="$(jq -r '.resource_version.id' "${WORK_DIR}/resource-upload.json")"

json_post "/api/v1/app/resource-check" \
  "{\"deviceId\":\"device-smoke\",\"appVersionCode\":100,\"resourceVersion\":\"20260601.1\",\"channel\":\"dev\"}" \
  | tee "${WORK_DIR}/resource-check.json" >/dev/null
jq -e --arg version "${RESOURCE_VERSION}" '.hasUpdate == true and .latestResourceVersion == $version and (.packages | length) > 0' \
  "${WORK_DIR}/resource-check.json" >/dev/null

json_post "/api/v1/app/update-event" \
  "{\"appKey\":\"game-helper-android\",\"deviceId\":\"device-smoke\",\"eventType\":\"resource_activation_failed\",\"toVersion\":\"${RESOURCE_VERSION}\",\"packageKey\":\"templates-common\",\"errorMessage\":\"smoke resource activation failed\"}" \
  | tee "${WORK_DIR}/resource-activation-failed.json" >/dev/null

curl -fsS "${BASE_URL}/admin/api/app-releases" | tee "${WORK_DIR}/overview-after.json" >/dev/null
jq -e --arg rid "${RESOURCE_ID}" '.resource_versions[] | select(.id == $rid and .status == "paused")' \
  "${WORK_DIR}/overview-after.json" >/dev/null
jq -e '.audit_logs[] | select(.action == "resource.auto_pause")' "${WORK_DIR}/overview-after.json" >/dev/null

cat <<EOF
release center DB smoke passed
- base_url: ${BASE_URL}
- database: ${DB_NAME}
- schema: ${DB_SCHEMA}
- release_id: ${RELEASE_ID}
- resource_id: ${RESOURCE_ID}
EOF
