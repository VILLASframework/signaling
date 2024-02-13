# SPDX-FileCopyrightText: 2023 OPAL-RT Germany GmbH
# SPDX-License-Identifier: Apache-2.0
{
  lib,
  buildGoModule,
  src,
}:
buildGoModule {
  pname = "villas-signaling-server";
  version = "master";
  src = src;
  vendorHash = "sha256-5up73J/bzMHlVlwke4w6eetXz7a+SYQJuQnTkP/L2JE=";
  meta = with lib; {
    mainProgram = "server";
    license = licenses.asl20;
  };
}
