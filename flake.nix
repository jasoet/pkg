{
  description = "Go pkg/v2 development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          packages = [
            # Go toolchain
            pkgs.go
            pkgs.golangci-lint
            pkgs.gofumpt
            pkgs.gosec

            # Container CLIs (daemon is system-level)
            pkgs.podman
            pkgs.docker-client
            pkgs.podman-compose

            # Other tools
            pkgs.bun
            pkgs.jq
          ];

          shellHook = ''
            echo "pkg/v2 dev environment ready — Go $(go version | awk '{print $3}')"
          '';
        };
      });
}
