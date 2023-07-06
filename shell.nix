{pkgs}:
pkgs.mkShellNoCC {
  nativeBuildInputs = with pkgs; [
    act
    deadnix
    gci
    git
    go_1_20
    gofumpt
    golangci-lint
    govulncheck
    shellcheck
    statix
    yamllint
  ];
}
