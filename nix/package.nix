{
  pkgs,
  flake,
  ...
}:
pkgs.buildGoModule {
  pname = "shellhubctl";
  version = "0.1.0";

  src = flake;

  vendorHash = "sha256-BqhwyKE8V547sC6DYuPmr1+lsOmkqkekssAtpAr7dhw=";
  goSum = flake;

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
