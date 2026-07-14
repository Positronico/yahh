#!/usr/bin/env bash
# End-to-end test: drives an interactive zsh in a sandboxed HOME and checks
# that commands land in the right history file as directories change.
set -euo pipefail
cd "$(dirname "$0")/.."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

go build -o "$TMP/yahh" .

export HOME="$TMP/home"
export ZDOTDIR="$HOME"
export YAHH_DATA_DIR="$HOME/.local/share/yahh"
unset HISTFILE
mkdir -p "$HOME/proj/sub" "$HOME/other"

cat > "$HOME/.zshrc" <<RC
HISTFILE="\$HOME/.zsh_history"
HISTSIZE=1000
SAVEHIST=1000
eval "\$("$TMP/yahh" init zsh --no-autoclean --no-completions)"
RC

LOG="$TMP/session.log"

run_zsh() {
  zsh -i > "$LOG" 2>&1 || true
}

fail() {
  echo "FAIL: $1" >&2
  echo "--- session log ---" >&2
  cat "$LOG" >&2 || true
  echo "--- global history ---" >&2
  cat "$HOME/.zsh_history" >&2 || true
  echo "--- realm histories ---" >&2
  cat "$YAHH_DATA_DIR"/histories/* >&2 || true
  exit 1
}

# --- Scenario 1: create a realm, verify history separation -----------------
run_zsh <<'EOF'
cd ~/proj
yahh create --name proj
true marker-in-realm
cd ~/proj/sub
true marker-in-subdir
cd ~
true marker-global
exit
EOF

realm_file="$YAHH_DATA_DIR/histories/1-proj.zsh.history"
[ -f "$realm_file" ] || fail "realm history file not created"
grep -q "marker-in-realm" "$realm_file" || fail "realm command missing from realm history"
grep -q "marker-in-subdir" "$realm_file" || fail "subdirectory command missing from realm history"
grep -q "marker-in-realm" "$HOME/.zsh_history" && fail "realm command leaked into global history"
grep -q "marker-global" "$HOME/.zsh_history" || fail "global command missing from global history"
grep -q "marker-global" "$realm_file" && fail "global command leaked into realm history"

# --- Scenario 2: realm history is recalled on re-entry ---------------------
run_zsh <<'EOF'
cd ~/proj
fc -l 1 > ~/recalled.txt
exit
EOF
grep -q "marker-in-realm" "$HOME/recalled.txt" || fail "realm history not loaded on re-entry"
grep -q "marker-global" "$HOME/recalled.txt" && fail "global history visible inside realm"

# --- Scenario 3: disable falls back to global history ----------------------
run_zsh <<'EOF'
yahh disable
cd ~/proj
true marker-disabled
cd ~
yahh enable
exit
EOF
grep -q "marker-disabled" "$HOME/.zsh_history" || fail "disabled-mode command missing from global history"
grep -q "marker-disabled" "$realm_file" && fail "disabled-mode command leaked into realm history"

# --- Scenario 4: remove while inside returns to global ---------------------
run_zsh <<'EOF'
cd ~/proj
yahh remove --yes
true marker-after-remove
exit
EOF
grep -q "marker-after-remove" "$HOME/.zsh_history" || fail "post-remove command missing from global history"
[ -f "$realm_file" ] && fail "realm history file still live after remove"
ls "$YAHH_DATA_DIR"/archive/1-proj.zsh.history.removed.* >/dev/null 2>&1 || fail "realm history not archived"

# --- Scenario 5: clean flags realms whose directory vanished ---------------
mkdir -p "$HOME/doomed"
"$TMP/yahh" create --name doomed "$HOME/doomed" >/dev/null
rmdir "$HOME/doomed"
"$TMP/yahh" clean --yes >/dev/null
"$TMP/yahh" list | grep -q "orphaned" || fail "vanished directory not flagged as orphaned"

echo "e2e zsh: OK"
