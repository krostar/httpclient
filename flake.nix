{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:krostar/nixpkgs/gci";
  };

  outputs = {
    self,
    flake-utils,
    nixpkgs,
    ...
  }:
    flake-utils.lib.eachSystem (flake-utils.lib.defaultSystems ++ [flake-utils.lib.system.x86_64-darwin]) (
      system: let
        pkgs = import nixpkgs {
          inherit system;
        };
      in {
        devShells.default = import ./shell.nix {inherit pkgs;};
      }
    );
}
