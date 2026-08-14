{
  pkgs,
  flake,
  pname,
  perSystem,
  ...
}:
perSystem.self.default.overrideAttrs (old: {
  inherit pname;

  nativeBuildInputs = (old.nativeBuildInputs or [ ]) ++ [ pkgs.golangci-lint ];

  buildPhase = ''
    runHook preBuild
    export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint
    golangci-lint run ./...
    runHook postBuild
  '';

  checkPhase = "true";

  installPhase = ''
    runHook preInstall
    touch $out
    runHook postInstall
  '';

  meta = (old.meta or { }) // {
    description = "golangci-lint over ${flake.shortRev or "workdir"}";
  };
})
