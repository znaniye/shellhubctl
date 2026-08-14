{ pname, pkgs, ... }:
pkgs.writeShellApplication {
  name = pname;

  runtimeInputs = [
    pkgs.deadnix
    pkgs.nixfmt
    pkgs.gofumpt
  ];

  text = ''
    set -euo pipefail

    if [[ $# = 0 ]]; then
      set -- "$(git rev-parse --show-toplevel 2>/dev/null || echo .)"
    fi

    deadnix --no-lambda-pattern-names --edit "$@"
    git ls-files -z "$@" | grep --null '\.nix$' | xargs --null --no-run-if-empty nixfmt
    gofumpt -l -w "$@"
  '';

  meta.description = "format the shellhub-tui project (nix + go)";
}
