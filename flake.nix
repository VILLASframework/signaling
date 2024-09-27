# SPDX-FileCopyrightText: 2023 OPAL-RT Germany GmbH
# SPDX-License-Identifier: Apache-2.0
{
  description = "a tool for connecting real-time power grid simulation equipment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }@inputs:
    {
      nixosModules.default =
        {
          pkgs,
          config,
          lib,
          ...
        }:
        let
          cfg = config.services.villas-signaling;
        in
        {
          options.services.villas-signaling = {
            enable = lib.mkEnableOption "VILLASnode WebRTC signaling server";

            address = lib.mkOption {
              description = "HTTP service address";
              default = ":8080";
              type = lib.types.str;
            };

            level = lib.mkOption {
              description = "The log level";
              default = "debug";
              type = lib.types.str;
            };

            relays = lib.mkOption {
              description = "A TURN/STUN relay which is signalled to each connection";
              type = lib.types.listOf lib.types.str;
              default = [ ];
            };

            api = {
              username = lib.mkOption {
                description = "Username for API endpoint";
                default = "admin";
                type = lib.types.str;
              };
              password = lib.mkOption {
                description = "Password for API endpoint";
                type = lib.types.str;
                default = "";
              };
              token = lib.mkOption {
                description = "Bearer token for authentication";
                type = lib.types.str;
                default = "";
              };
            };
          };

          config = {
            nixpkgs.overlays = [
              self.overlays.default
            ];

            systemd.services.villas-signaling-server = lib.mkIf cfg.enable {
              enable = true;
              description = "VILLASnode WebRTC Signaling Server";
              serviceConfig = {
                Type = "simple";
                ExecStart =
                  let
                    args = lib.cli.toGNUCommandLine { mkOptionName = k: "-${k}"; } {
                      inherit (cfg) address level;
                      relay = cfg.relays;
                      api-username = if cfg.api.username != "" then cfg.api.username else null;
                      api-password = if cfg.api.password != "" then cfg.api.password else null;
                      api-token = if cfg.api.token != "" then cfg.api.token else null;
                    };
                    argsStr = builtins.concatStringsSep " " args;
                  in
                  "${pkgs.villas-signaling}/bin/server ${argsStr}";
              };
              wantedBy = [ "multi-user.target" ];
            };
          };
        };

      overlays.default = final: prev: { villas-signaling = final.callPackage ./default.nix { }; };
    }
    // flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ self.overlays.default ];
        };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;
        apps =
          let
            inherit (self.packages.${system}) server;
          in
          {
            turn-api-auth = {
              type = "app";
              program = "${pkgs.villas-signaling}/bin/turn-api-auth";
            };
          };

        packages = rec {
          default = pkgs.villas-signaling;
        };

        devShells = {
          default = pkgs.mkShell { inputsFrom = [ pkgs.villas-signaling ]; };
        };

        checks = {
          fmt = pkgs.runCommand "check-fmt" { } ''
            cd ${self}
            ${self.formatter.${system}}/bin/nixfmt --check . 2> $out
          '';
        };
      }
    );
}
