```
██╗   ██╗ █████╗ ██╗  ██╗██╗  ██╗
╚██╗ ██╔╝██╔══██╗██║  ██║██║  ██║
 ╚████╔╝ ███████║███████║███████║
  ╚██╔╝  ██╔══██║██╔══██║██╔══██║
   ██║   ██║  ██║██║  ██║██║  ██║
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝
```
# YAHH (Yet Another History Hack)

YAHH keeps a **separate shell command history per project**. Register a
directory as a **realm** and every shell session inside that directory tree
records to — and recalls from — the realm's own history file. Leave the tree
and your global history returns, automatically.

Useful when you context-switch between many projects (consulting,
professional services, ops): the build/test/deploy incantations of each
project stay together, without drowning your global history.

![YAHH demo](demo.gif)

Works with **zsh** and **bash**.

---

## How it works

- The `yahh` binary keeps a central **SQLite registry** mapping directories
  to realms (`~/.local/share/yahh/yahh.db`). No marker files are dropped
  into your projects.
- Each realm has its own history files under `~/.local/share/yahh/histories/`,
  in your shell's **native format** — the shell itself reads and writes them
  with its normal history machinery.
- A small hook (installed with one `eval` line) detects directory changes
  and swaps `HISTFILE` accordingly. A realm covers its whole subtree; the
  deepest matching realm wins, so realms can nest.
- Because zsh and bash history formats are incompatible, each realm keeps
  one file per shell. zsh and bash sessions in the same realm therefore
  have separate histories.

## Install

### Homebrew

```sh
brew install Positronico/tap/yahh
```

### From source

```sh
go install github.com/Positronico/yahh@latest
```

### Activate the shell integration

Either let yahh edit your rc file(s):

```sh
yahh install        # detects zsh/bash, appends a guarded eval block
```

or add the line yourself:

```sh
eval "$(yahh init zsh)"     # ~/.zshrc
eval "$(yahh init bash)"    # ~/.bashrc
```

Then restart your shell. Tab completion is set up by the same line
(Homebrew also installs completions natively).

## Usage

```sh
cd ~/work/big-project
yahh create                 # this tree is now the realm "big-project"
yahh create --name api --import   # named realm, seeded with your last 1000 global entries
```

| Command | What it does |
|---|---|
| `yahh create [dir]` | Register a realm. `--name X`, `--import[=N]` (seed from global history), `--from FILE`, `--force`. |
| `yahh remove [dir]` | Unregister the realm covering `dir` (or `--name X`). History is archived; `--merge` folds it into your global history first (`--into FILE` to override the target), `--purge` deletes it. |
| `yahh list` | All realms with path, entry count, last use, orphan state. `--json`, `--paths`. |
| `yahh which [dir]` | Show which realm covers a directory. |
| `yahh search TERM` | Search across all realm histories. `--realm X`, `--regex`, `--global`, `--json`. |
| `yahh mv REALM DIR` | Re-point a realm after moving the project directory. |
| `yahh rename REALM NAME` | Rename a realm. |
| `yahh clean` | Flag realms whose directory vanished; remove them after a grace period (default 30d). `--dry-run`, `--yes`, `--grace 30d`, `--purge-archive[=90d]`. |
| `yahh enable` / `disable` | Turn realm switching on/off globally (persisted; applies on each shell's next `cd`). |
| `yahh status` | Current state and the realm covering the current directory. |
| `yahh install` / `uninstall` | Manage the rc-file integration. `uninstall --purge` also deletes all data. |
| `yahh completion <shell>` | Print the completion script (zsh, bash, fish, powershell). |

### Cleanup

Deleted a project without removing its realm? Nothing rots:

- On shell startup, yahh triggers a **throttled background clean** (at most
  once a week) that flags realms whose directories no longer exist.
- A flagged realm is left alone for a **grace period** (30 days by default —
  it might be an unmounted volume). If the directory comes back, the flag is
  cleared; if not, the realm is removed and its history **archived** to
  `~/.local/share/yahh/archive/`.
- `yahh clean --purge-archive` deletes old archives when you're sure.

Removing a realm never silently destroys history: it is archived unless you
pass `--purge`.

### Escape hatch

Set `YAHH_DISABLE=1` in a session to bypass realm switching entirely for
that shell, or run `yahh disable` to turn it off everywhere.

## Notes for zsh users

yahh no longer forces any `setopt` on you (v1 set six). Recommended options
that play well with per-realm histories:

```zsh
setopt HIST_IGNORE_ALL_DUPS HIST_IGNORE_SPACE HIST_FIND_NO_DUPS
```

`SAVEHIST` must be greater than 0 for histories to be written at all.
`SHARE_HISTORY` and `INC_APPEND_HISTORY` work, but cross-shell sharing is
scoped to whatever realm each shell is currently in.

## Data layout

```
~/.local/share/yahh/            # or $XDG_DATA_HOME/yahh, or $YAHH_DATA_DIR
├── yahh.db                     # realm registry (SQLite)
├── histories/                  # live realm histories (<id>-<name>.<shell>.history)
└── archive/                    # archived histories from remove/clean
```

Histories are created mode 0600 — they can contain secrets.

## Migrating from v1

v2 is a fresh start: the old `.history` pointer files and
`~/.config/yahh/realm_*.history` files are not read. To carry a v1 realm
over:

```sh
cd ~/that/project
yahh create --from ~/.config/yahh/realm_<hash>.history
```

Then delete the project's `.history` pointer file and remove the old
`source .../yahh.zsh` line from your `~/.zshrc` (the v1 script lives in
`legacy/` for one more release).

## Changelog

- **v2.0.0** — Rewritten in Go. bash support, central SQLite registry (no
  more in-repo pointer files), named realms, history import/merge,
  cross-realm search, automatic orphan cleanup, self-install, shell
  completions, Homebrew tap.
- **v1.0.0** — Initial release (zsh script).

## Contributing

Issues and pull requests are welcome. Run the tests with:

```sh
go test ./...
./test/e2e_zsh.sh
./test/e2e_bash.sh
```

## License

[MIT](LICENSE)

---

*Happy hacking with YAHH!*
