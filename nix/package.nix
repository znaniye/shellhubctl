{
  pkgs,
  flake,
  ...
}:
pkgs.buildGoModule {
  pname = "shellhubctl";
  version = "0.1.0";

  src = flake;

  vendorHash = "sha256-3yZK7hDeYY7AYnqf1WalpG8P0At3FRZMaPloQctp5ac=";

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Terminal UI for ShellHub";
    mainProgram = "shellhubctl";
  };
}
