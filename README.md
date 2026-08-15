# shellhubctl

A terminal UI and a JSON-emitting CLI for [ShellHub](https://shellhub.io).

![shellhubctl](docs/demo.gif)

Two faces, one binary:

- run it with no arguments and you get an interactive TUI — log in, pick a
  namespace, browse devices, open an SSH session;
- run `shellhubctl devices` and you get a JSON array on stdout, meant to be piped
  into `jq` and consumed by whatever automation you already have.

The JSON side is deliberately unopinionated. It returns the devices as the
ShellHub API describes them and gets out of the way; it does not know or care
whether you are deploying NixOS, running Ansible, or writing a shell loop.

## Install

With Nix, from a clone:

```sh
nix run . -- devices
nix profile install .
```

From source:

```sh
go build -o shellhubctl .
```

## Choosing a server

Every command resolves the server in this order:

1. the `--server` flag
2. the `SHELLHUB_URL` environment variable
3. `https://cloud.shellhub.io`

## The terminal UI

```sh
shellhubctl
```

Authentication is by username (or email) and password. The resulting JWT is
stored at `$XDG_CONFIG_HOME/shellhubctl/session.json` with mode `0600`, and is
reused until it expires.

ShellHub has no token revocation endpoint, so `shellhubctl logout` only deletes
the local file — the token itself stays valid on the server until it expires.

```sh
shellhubctl logout
```

## The JSON CLI

```sh
shellhubctl devices
```

`devices` authenticates with an **API key**, never with the TUI's stored session.
The two credential paths are kept separate on purpose: a key is scoped to a
single namespace by the server and is what you want in a script, while the
session is interactive and expires.

### Providing the API key

There is no `--api-key` flag, and there will not be one: anything in `argv` is
readable by every other user on the machine via `ps`. The key comes from the
environment, in this order:

| Variable | Meaning |
|---|---|
| `SHELLHUB_API_KEY` | the key itself |
| `SHELLHUB_API_KEY_COMMAND` | a shell command whose stdout is the key |
| `SHELLHUB_API_KEY_FILE` | a file containing the key |

The first one that is set and non-empty wins; surrounding whitespace is stripped.
A variable that is set but empty is treated as unset and the next source is
tried. A command that *fails*, however, is an error rather than a fallback — if
you said the key comes from there, silently looking elsewhere would hide the
problem.

`SHELLHUB_API_KEY_COMMAND` is the one to reach for with a password manager:

```sh
export SHELLHUB_API_KEY_COMMAND='pass show shellhub/api-key'
export SHELLHUB_API_KEY_COMMAND='op read op://private/shellhub/api-key'
export SHELLHUB_API_KEY_COMMAND='sops decrypt --extract "[\"api_key\"]" secrets.yaml'
```

The key never appears in output, in an error message, or in a log. If your
command fails, its *stderr* is shown so you can debug it — its stdout, which is
the key, is not.

### Selecting devices

```sh
shellhubctl devices --tag builder --online
shellhubctl devices --distro 'nixos' --name '^web-'
shellhubctl devices --name 'web' --name 'prod'
```

| Flag | Effect |
|---|---|
| `--name` | regular expression matched against the device name |
| `--distro` | regular expression matched against `info.pretty_name` |
| `--tag` | exact tag name |
| `--online` | keep only devices currently online |
| `--status` | device status to list, defaults to `accepted` |
| `--ssh-user` | user used to build `sshid`, defaults to `$SHELLHUB_SSH_USER` then `root` |

`--name`, `--distro` and `--tag` are repeatable, and **every** selector must
match — they combine with AND, not OR. Regular expressions are Go's RE2, applied
case-insensitively and unanchored, so `--name web` matches `my-web-01`. Inline
flags work: `--name '(?-i)Web'` forces a case-sensitive match.

All pages are fetched, so the output is the whole namespace, not the first
hundred devices. Results are sorted by name.

### Output

A JSON array on stdout. Each element is the device exactly as the API returns
it, with two additions:

- `tags` is flattened from ShellHub's tag objects to a plain list of names;
- `sshid` is derived as `<ssh-user>@<namespace>.<device>@<gateway>`.

```json
[
  {
    "uid": "a1b2c3d4e5f6",
    "name": "builder-01",
    "online": true,
    "status": "accepted",
    "namespace": "acme",
    "tenant_id": "00000000-0000-0000-0000-000000000000",
    "last_seen": "2026-08-15T12:30:00Z",
    "created_at": "2026-01-04T09:12:00Z",
    "info": {
      "id": "nixos",
      "pretty_name": "NixOS 25.05 (Warbler)",
      "version": "v0.18.4",
      "arch": "x86_64",
      "platform": "docker"
    },
    "identity": { "mac": "02:42:ac:11:00:02" },
    "tags": ["builder", "amd64"],
    "sshid": "root@acme.builder-01@cloud.shellhub.io"
  }
]
```

An empty result is an empty array and exit status 0 — matching nothing is an
answer, not a failure. Errors go to stderr and exit 1, so stdout is always
either valid JSON or nothing at all.

### Piping it somewhere

```sh
shellhubctl devices --tag builder | jq -r '.[].sshid'

shellhubctl devices --online --tag web \
  | jq -r '.[].sshid' \
  | xargs -P8 -I{} ssh {} 'systemctl restart nginx'

shellhubctl devices | jq -r '.[] | select(.info.arch == "aarch64") | .name'
```

## Development

The project uses Nix. `nix develop` gives you Go, `golangci-lint` and `gofumpt`.

```sh
go build ./...
go test ./...
golangci-lint run ./...
gofumpt -l main.go internal

nix flake check
nix build .#default
```
