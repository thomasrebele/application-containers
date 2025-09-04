

- check https://github.com/mviereck/x11docker

- run an intermediate wayland server (e.g., sommelier)?

- Intellij
  - provide environment with required libraries: nix-shell intellij-fhsenv.nix
  - some settings that may or may not be necessary
      LD_LIBRARY_PATH=/usr/lib64/ DISPLAY=:0 /home/tr/software/intellij/idea-IC-251.26927.53/bin/idea
      export LD_LIBRARY_PATH=/usr/lib:/lib:/lib64:$LD_LIBRARY_PATH
      export PATH=$PATH:~/.jdks/ms-17.0.15/bin/
      export JAVA_HOME=~/.jdks/ms-17.0.15
      export GRADLE_USER_HOME=$HOME/.gradle
  - needed to rename /home/act-intellij-experimental/.jdks/ms-17.0.15/bin/java to java2 and provide a script
    #!/bin/sh
    export LD_LIBRARY_PATH=/lib64:$LD_LIBRARY_PATH
    /home/act-intellij-experimental/.jdks/ms-17.0.15/bin/java2 "$@"
  - FYI, env:
    FONTCONFIG_PATH=/etc/fonts
    JAVA_HOME=/home/act-intellij-experimental/.jdks/ms-17.0.15
    PWD=/
    container=podman
    HOME=/home/act-intellij-experimental
    WAYLAND_DISPLAY=wayland-1
    GRADLE_USER_HOME=/home/act-intellij-experimental/.gradle
    TERM=xterm
    SHLVL=1
    LD_LIBRARY_PATH=/usr/lib:/lib:/lib64:
    XDG_RUNTIME_DIR=/run/user/1001
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/act-intellij-experimental/.jdks/ms-17.0.15/bin/:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
    PULSE_SERVER=unix:/run/user/1001/pulse/native
    OLDPWD=/tmp
    _=/usr/bin/env
