# STK — Sigma Token Killer

STK is a stdlib-only Go token-reducer. It is **NOT rtk** and has no relation to
it. STK works on Linux, macOS, and Windows. Usage is explicit opt-in only.

STK is open source (MIT). The public repo is
<https://github.com/sigmashakeinc/stk>. The SigmaShake monorepo is the source of
truth. A mirror script syncs the monorepo to the public repo.

## What it does

STK wraps any dev command, captures its combined stdout+stderr, applies a
deterministic reduction pipeline, and prints the condensed output — saving AI
agent context tokens without losing signal.

Reduction pipeline (applied in order):

1. **CR strip** — progress bars and spinners overwrite lines with `\r`; STK
   keeps only the final segment after the last carriage return.
2. **Consecutive-duplicate collapse** — a run of N identical lines (N>1)
   becomes one copy with a ` (×N)` suffix.
3. **Blank-run collapse** — consecutive blank/whitespace-only lines collapse to
   a single blank line.
4. **Importance-preserving truncation** — if the line count still exceeds
   `--max` (default 200), STK always keeps lines matching
   `(?i)(error|warn|fail|fatal|panic|denied|refus|exception|cannot|undefined)`,
   then fits the remaining lines into a head block (~2/3) and a tail block
   (~1/3), inserting a single `… [stk] trimmed <M> line(s) …` marker. Error
   lines are **never dropped**.

STK does **NOT** register a silent global command-rewrite hook (unlike rtk).
Usage is always explicit — `stk <cmd>` — and an SSG LOG nudge will remind you
if you forget.

## Install

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/sigmashakeinc/stk/main/install.sh | sh
```

Windows (PowerShell):

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/sigmashakeinc/stk/main/install.ps1 | iex"
```

Go:

```sh
go install github.com/sigmashakeinc/stk@latest
```

The installers download the release archive and `checksums.txt`. They verify
the SHA-256 hash. They stop if the hash does not match.

| Setting | Linux/macOS | Windows |
|---|---|---|
| Install directory | `INSTALL_DIR` (default `~/.local/bin`) | `-InstallDir` or `INSTALL_DIR` (default `%LOCALAPPDATA%\Programs\stk`) |
| Release tag | `VERSION` or `--version` | `-Version` or `VERSION` |
| Local archive | `--from-file` (and `--checksums`) or `STK_FROM_FILE` | `-FromFile` (and `-ChecksumsFile`) or `STK_FROM_FILE` |

The Windows installer adds the install directory to your user PATH. It needs
no admin rights.

Build from source:

```sh
go build -ldflags "-X main.version=dev" -o stk .
```

## Usage

```sh
# Reduce go build output:
stk go build ./...

# Reduce npm install chatter:
stk --max 50 npm install

# Passthrough (no reduction — useful in scripts that need raw output):
stk --raw go test ./...

# Print version:
stk --version
stk version
```

## Flags and env vars

| Flag / Env | Default | Description |
|---|---|---|
| `--max <n>` | 200 | Max output lines before truncation |
| `STK_MAX_LINES=<n>` | 200 | Same as `--max`; flag takes precedence |
| `--raw` | off | Passthrough mode, no capture |
| `STK_DISABLE=1` | off | Same as `--raw` |

STK stops parsing its own flags at the first non-STK token, so flags belonging
to the wrapped command are always passed through verbatim:

```sh
stk git log --oneline   # --oneline goes to git, not stk
stk go test -v ./...    # -v goes to go test
```

## Exit codes

STK preserves the child process exit code exactly, so `stk <cmd> && next` chains
work correctly. Exit 127 means the command could not be started (not found, no
permission, etc.).

## Running tests

```sh
cd sigmashake-stk
bun run gate:test go test -cover ./...
```

## Do not reintroduce rtk

`rtk` is intentionally absent from this repo. Do not add it as a dependency.
Do not import it in any script. STK never installs a silent command-rewrite
hook.

## License

MIT. See `LICENSE`.
