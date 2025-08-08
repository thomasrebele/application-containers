{ pkgs ? import <nixpkgs> {} }:

(pkgs.buildFHSEnv {
  name = "intellij-env";
  targetPkgs = pkgs: (with pkgs; [
    zlib
    strace
    fontconfig
    udev
  ]) ++ (with pkgs.xorg; [
    libX11
    libXext
    libXrender
    libXtst
    libXi
  ]);
  multiPkgs = pkgs: (with pkgs; [
  ]);
  runScript = "bash";
}).env
