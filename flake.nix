{
  description = "An example project using flutter";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.gomod2nix = {
        url = "github:tweag/gomod2nix";
        inputs.nixpkgs.follows = "nixpkgs";
        inputs.utils.follows = "utils";
      };
  outputs = {
    self,
    flake-utils,
    nixpkgs,
    gomod2nix,
    ...
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ gomod2nix.overlays.default ];
      };
    in {
      devShell = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gotools
          go-tools
          gomod2nix.packages.${system}.default
          sqlite-interactive
        ];
      };
    });
}
