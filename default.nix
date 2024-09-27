# SPDX-FileCopyrightText: 2023 OPAL-RT Germany GmbH
# SPDX-License-Identifier: Apache-2.0
{
  lib,
  buildGoModule,
}:
buildGoModule {
  pname = "villas-signaling";
  version = "master";
  src = ./.;
  vendorHash = "sha256-oGkooQbRk9EwMOQNxIcaIGX/jo1df5gY9KsiFVUALKE=";
  meta = with lib; {
    mainProgram = "server";
    license = licenses.asl20;
  };
}
