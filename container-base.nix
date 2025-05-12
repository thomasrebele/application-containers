{ pkgs ? import <nixpkgs> { }
, pkgsLinux ? import <nixpkgs> { system = "x86_64-linux"; }
}:
let
  addendum = pkgs.lib.fileset.toSource {
    root = ./addendum;
    fileset = ./addendum;
  };
in 
with pkgsLinux; pkgs.dockerTools.buildImage {
  name = "thinbase";
  tag = "latest";
  copyToRoot = [
    fakeNss
    bashInteractive
    coreutils
    busybox
    dbus
    addendum
  ];
  config = {
    Cmd = [ "${pkgsLinux.bashInteractive}/bin/bash" ];
  };
}

