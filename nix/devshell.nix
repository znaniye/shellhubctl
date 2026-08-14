{ pkgs, perSystem, ... }:
pkgs.mkShell {
  packages = [
    pkgs.go
    pkgs.gopls
    pkgs.gotools
    pkgs.go-tools
    pkgs.delve

    pkgs.golangci-lint
    pkgs.gofumpt
    perSystem.self.formatter

    pkgs.curl
    pkgs.jq
    pkgs.websocat
  ];

  env = {
    GOFLAGS = "-mod=mod";
    CGO_ENABLED = "0";
  };

  shellHook = ''
    export PRJ_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
    export GOPATH="$PRJ_ROOT/.direnv/go"
    export GOBIN="$GOPATH/bin"
    export PATH="$GOBIN:$PATH"
  '';
}
