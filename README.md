
Build and install the image

$ nix-build container-base.nix 
/nix/store/416fg796534751lds7n2cqi3wjfg7jv4-docker-image-thinbase.tar.gz

$ podman load < /nix/store/416fg796534751lds7n2cqi3wjfg7jv4-docker-image-thinbase.tar.gz
Getting image source signatures
Copying blob 1a6a1a647a52 done   | 
Copying config 416b498c4b done   | 
Writing manifest to image destination
Loaded image: localhost/thinbase:latest
