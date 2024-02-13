# SPDX-FileCopyrightText: 2023 OPAL-RT Germany GmbH
# SPDX-License-Identifier: Apache-2.0
{
  description = "a tool for connecting real-time power grid simulation equipment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  } @ inputs: let
    inherit (nixpkgs) lib;
    supportedSystems = ["x86_64-linux" "aarch64-linux"];
    forSupportedSystems = lib.genAttrs supportedSystems;
  in {
    formatter = forSupportedSystems (system: nixpkgs.legacyPackages.${system}.alejandra);
    apps = forSupportedSystems (
      system: let
        inherit (self.packages.${system}) server;
      in {
        turn-api-auth = {
          type = "app";
          program = "${server}/bin/turn-api-auth";
        };
      }
    );
    packages = forSupportedSystems (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in rec {
        default = server;
        server = pkgs.callPackage ./packaging/nix/server.nix {
          src = ./.;
        };
      }
    );
    devShells = forSupportedSystems (
      system: let
        shellHook = ''[ -z "$PS1" ] || exec $SHELL'';
        pkgs = nixpkgs.legacyPackages.${system};
        packages = [pkgs.bashInteractive];
      in {
        default = pkgs.mkShell {
          inherit shellHook packages;
          name = "villas-signaling-server";
          inputsFrom = [self.packages.${system}.server];
        };
      }
    );
    checks = forSupportedSystems (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        fmt = pkgs.runCommand "check-fmt" {} ''
          cd ${self}
          ${pkgs.alejandra}/bin/alejandra --check . 2> $out
        '';
      }
    );
  };
}
