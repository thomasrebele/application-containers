{ pkgs ? import <nixpkgs> { }
, pkgsLinux ? import <nixpkgs> { system = "x86_64-linux"; }
}:
pkgs.dockerTools.buildImage {
  name = "thinbase";
  tag = "latest";
  copyToRoot = pkgs.buildEnv {
    name = "image-root";
    paths = with pkgsLinux; [
      bashInteractive
      coreutils
      busybox
    ];
    pathsToLink = [ "/bin" ];
  };
  config = {
    Cmd = [ "${pkgsLinux.bashInteractive}/bin/bash" ];
  };
}

