{ pkgs ? import <nixpkgs> {}}:
pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
    bashInteractive
    git
    go_1_19
    golangci-lint
    gofumpt
    gci
    shellcheck
    yamllint
  ];
}
