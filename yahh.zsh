# YAHH – Yet Another History Hack

export YAHH_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/yahh"
export YAHH_CONFIG_FILE="${YAHH_CONFIG_DIR}/config.conf"
: ${DEFAULT_HISTFILE:="$HOME/.zsh_history"}

mkdir -p "$YAHH_CONFIG_DIR"

# Load persistent config if available
if [[ -f "$YAHH_CONFIG_FILE" ]]; then
  source "$YAHH_CONFIG_FILE"
fi
if [[ -z "$YAHH_ENABLED" ]]; then
  YAHH_ENABLED=1
fi

function yahh_save_config() {
  echo "YAHH_ENABLED=$YAHH_ENABLED" > "$YAHH_CONFIG_FILE"
}

# A helper function to switch history files:
function yahh_switch_history() {
  local new_file="$1"
  fc -W  # flush current in-memory history to the current file
  if [[ "$HISTFILE" != "$new_file" ]]; then
    HISTFILE="$new_file"
    [[ ! -f "$HISTFILE" ]] && touch "$HISTFILE"
    # Push the new history file and clear the in-memory history list.
    fc -p "$HISTFILE"
    # If the new file already has content, load it.
    if [[ -s "$HISTFILE" ]]; then
      fc -R "$HISTFILE"
    fi
  fi
}

# Search upward from $PWD for a .history pointer file (stop at $HOME)
function yahh_find_realm() {
  # Skip search if YAHH is disabled.
  if (( ! YAHH_ENABLED )); then
    return 1
  fi
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/.history" ]]; then
      echo "$dir/.history"
      return 0
    fi
    if [[ "$dir" == "$HOME" ]]; then
      break
    fi
    dir=$(dirname "$dir")
  done
  return 1
}

# Return the actual history file path for the current realm or fallback
function yahh_get_histfile() {
  if (( YAHH_ENABLED )); then
    local realm_file
    realm_file=$(yahh_find_realm)
    if [[ -n "$realm_file" ]]; then
      local hist_path
      hist_path=$(head -n 1 "$realm_file")
      [[ -n "$hist_path" ]] && { echo "$hist_path"; return 0; }
    fi
  fi
  echo "$DEFAULT_HISTFILE"
}

# Update HISTFILE based on the current directory's realm (if enabled)
function yahh_update_history() {
  if (( ! YAHH_ENABLED )); then
    if [[ "$HISTFILE" != "$DEFAULT_HISTFILE" ]]; then
      yahh_switch_history "$DEFAULT_HISTFILE"
    fi
    return
  fi
  local new_hist
  new_hist=$(yahh_get_histfile)
  if [[ "$HISTFILE" != "$new_hist" ]]; then
    yahh_switch_history "$new_hist"
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook chpwd yahh_update_history
precmd() { yahh_update_history }

# YAHH command
function yahh() {
  local cmd=$1
  case "$cmd" in
    create)
      # Force-enable YAHH upon creation of a new realm.
      YAHH_ENABLED=1
      yahh_save_config
      if [[ -f "./.history" ]]; then
        echo "A .history file already exists here."
        return 1
      fi
      if command -v md5sum >/dev/null 2>&1; then
        local hash
        hash=$(echo -n "$PWD" | md5sum | awk '{print $1}')
      else
        local hash
        hash=$(date +%s)
      fi
      local realm_name="realm_${hash}.history"
      local central_hist="$YAHH_CONFIG_DIR/$realm_name"
      echo "$central_hist" > ./.history
      # Create a new empty central history file.
      : > "$central_hist"
      # Switch to the new (empty) history realm.
      yahh_switch_history "$central_hist"
      echo "Created new empty history realm for $(pwd)."
      ;;
    remove)
      if [[ ! -f "./.history" ]]; then
        echo "No .history file in the current directory."
        return 1
      fi
      local central_hist
      central_hist=$(head -n 1 ./.history)
      rm -f ./.history
      local ts
      ts=$(date +%Y%m%d%H%M%S)
      local new_name="${central_hist}.removed.${ts}"
      mv "$central_hist" "$new_name"
      echo "Removed history realm from $(pwd)."
      echo "Central history file renamed to: $new_name"
      ;;
    prune)
      local removed_files=("$YAHH_CONFIG_DIR"/realm_*.history.removed.*(N))
      if [[ ${#removed_files[@]} -eq 0 ]]; then
        echo "No removed history files to prune."
        return 0
      fi
      echo "The following removed history files will be deleted:"
      for f in "${removed_files[@]}"; do
        echo "  $f"
      done
      echo -n "Proceed with deletion? (y/N): "
      read answer
      if [[ "$answer" =~ ^[Yy]$ ]]; then
        for f in "${removed_files[@]}"; do
          rm -f "$f"
        done
        echo "Pruned removed history files."
      else
        echo "Prune cancelled."
      fi
      ;;
    which)
      # Output only the path of the active .history pointer file if YAHH is enabled.
      if (( ! YAHH_ENABLED )); then
        return 0
      fi
      local realm_file
      realm_file=$(yahh_find_realm)
      if [[ -n "$realm_file" ]]; then
        echo "$realm_file"
      fi
      ;;
    list)
      echo "Listing all history realms in $YAHH_CONFIG_DIR:"
      echo ""
      echo "Active realms:"
      local active_files=("$YAHH_CONFIG_DIR"/realm_*.history(N))
      if [[ ${#active_files[@]} -eq 0 ]]; then
        echo "  None"
      else
        for f in "${active_files[@]}"; do
          echo "  $f"
        done
      fi
      echo ""
      echo "Removed realms:"
      local removed_files=("$YAHH_CONFIG_DIR"/realm_*.history.removed.*(N))
      if [[ ${#removed_files[@]} -eq 0 ]]; then
        echo "  None"
      else
        for f in "${removed_files[@]}"; do
          echo "  $f"
        done
      fi
      ;;
    disable)
      YAHH_ENABLED=0
      yahh_save_config
      yahh_switch_history "$DEFAULT_HISTFILE"
      echo "YAHH disabled. Default history in use."
      ;;
    enable)
      YAHH_ENABLED=1
      yahh_save_config
      yahh_update_history
      echo "YAHH enabled."
      ;;
    status)
      echo "YAHH Status:"
      echo "  Enabled: $YAHH_ENABLED"
      echo "  Config file: $YAHH_CONFIG_FILE"
      echo "  Config directory: $YAHH_CONFIG_DIR"
      echo "  Default history file: $DEFAULT_HISTFILE"
      echo "  Current directory: $PWD"
      if (( ! YAHH_ENABLED )); then
        echo "  Active realm: None (YAHH is disabled)"
      else
        local realm_file
        realm_file=$(yahh_find_realm)
        if [[ -n "$realm_file" ]]; then
          local central_hist
          central_hist=$(head -n 1 "$realm_file")
          echo "  Active realm pointer file: $realm_file"
          echo "  Active central history file: $central_hist"
        else
          echo "  Active realm: None"
        fi
      fi
      echo "  Current HISTFILE: $HISTFILE"
      ;;
    *)
      echo "YAHH commands:"
      echo "  create   - Create a new empty history realm in the current directory (and enable YAHH)."
      echo "  remove   - Remove the history realm from the current directory (renames central file)."
      echo "  prune    - Delete removed (inactive) central history files after confirmation."
      echo "  list     - List all active and removed history realms in the config directory."
      echo "  which    - Output the path of the active .history pointer file (if any)."
      echo "  disable  - Disable YAHH (persisted across sessions, disables realm search)."
      echo "  enable   - Enable YAHH (persisted across sessions)."
      echo "  status   - Show current YAHH status."
      ;;
  esac
}
