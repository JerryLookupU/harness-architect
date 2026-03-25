#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/klein-install-smoke.XXXXXX")"
CODEX_HOME_DIR="${TMP_ROOT}/codex-home"

cleanup() {
  rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT

mkdir -p "${CODEX_HOME_DIR}"

cat > "${CODEX_HOME_DIR}/AGENTS.md" <<'EOF'
# Personal AGENTS

- retain this note
EOF

cat > "${CODEX_HOME_DIR}/config.toml" <<'EOF'
model = "gpt-5.4-mini"

[profiles.personal]
model = "gpt-5.4-mini"
approval_policy = "on-request"
sandbox_mode = "workspace-write"
EOF

run_install() {
  CODEX_HOME="${CODEX_HOME_DIR}" \
    "${REPO_ROOT}/install.sh" \
    klein-harness \
    markdown-fetch \
    generate-contributor-guide \
    --dest "${CODEX_HOME_DIR}/skills" \
    --bin-dir "${CODEX_HOME_DIR}/bin" \
    --no-shell-rc \
    >/dev/null
}

run_install
run_install

AGENTS_FILE="${CODEX_HOME_DIR}/AGENTS.md"
CONFIG_FILE="${CODEX_HOME_DIR}/config.toml"

[[ "$(grep -c '^<!-- klein-harness managed global instructions:start -->$' "${AGENTS_FILE}")" -eq 1 ]]
[[ "$(grep -c '^<!-- klein-harness managed global instructions:end -->$' "${AGENTS_FILE}")" -eq 1 ]]
[[ "$(grep -c 'Prefer `jq`' "${AGENTS_FILE}")" -eq 1 ]]
grep -Fq '# Personal AGENTS' "${AGENTS_FILE}"
grep -Fq 'retain this note' "${AGENTS_FILE}"

[[ "$(grep -c '^# >>> klein-harness managed codex profiles >>>$' "${CONFIG_FILE}")" -eq 1 ]]
[[ "$(grep -c '^# <<< klein-harness managed codex profiles <<<$' "${CONFIG_FILE}")" -eq 1 ]]
[[ "$(grep -c '^\[profiles."klein-worker"\]$' "${CONFIG_FILE}")" -eq 1 ]]
grep -Fq 'model = "gpt-5.4-mini"' "${CONFIG_FILE}"
grep -Fq '[profiles.personal]' "${CONFIG_FILE}"
grep -Fq '[profiles."klein-orchestrator"]' "${CONFIG_FILE}"
grep -Fq '[profiles."klein-research"]' "${CONFIG_FILE}"

test -f "${CODEX_HOME_DIR}/skills/markdown-fetch/SKILL.md"
test -f "${CODEX_HOME_DIR}/skills/generate-contributor-guide/SKILL.md"
test -f "${CODEX_HOME_DIR}/skills/generate-contributor-guide/references/analysis-checklist.md"
grep -Fq '.harness/research/<slug>.md' "${CODEX_HOME_DIR}/skills/markdown-fetch/SKILL.md"
grep -Fq '.harness/research/contributor-guide.md' "${CODEX_HOME_DIR}/skills/generate-contributor-guide/SKILL.md"
grep -Fq '.harness/standards.md' "${CODEX_HOME_DIR}/skills/generate-contributor-guide/SKILL.md"
grep -Fq 'AGENTS.md' "${CODEX_HOME_DIR}/skills/generate-contributor-guide/SKILL.md"

for command_name in harness-submit harness-tasks harness-task harness-control; do
  [[ -x "${CODEX_HOME_DIR}/bin/${command_name}" ]]
done

echo "install managed home smoke passed"
