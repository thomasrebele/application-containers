{ pkgs ? import <nixpkgs> { }
, pkgsLinux ? import <nixpkgs> { system = "x86_64-linux"; }
}:
let
  addendum = pkgs.lib.fileset.toSource {
    root = ./addendum;
    fileset = ./addendum;
  };
  myFakeNss = pkgsLinux.fakeNss.override {
    extraPasswdLines = [''
      act-user:x:999:999::/home/act-user:/bin/sh
    ''];
    extraGroupLines = [''
      act-group:x:999:
    ''];
  };
in 
with pkgsLinux; pkgs.dockerTools.buildImage {
  name = "thinbase";
  tag = "latest";
  copyToRoot = [
    myFakeNss
    bashInteractive
    coreutils
    busybox
    dbus
    addendum
    shadow
  ];
  # do not use useradd, as /etc/passwd and /etc/group are managed by fakeNss
  runAsRoot = ''
    ${pkgs.dockerTools.shadowSetup}
    mkdir -p /home/act-user
    chown act-user:act-group /home/act-user
    chmod a+rw /tmp
  '';
  config = {
    Cmd = [ "${pkgsLinux.bashInteractive}/bin/bash" ];
  };
}

