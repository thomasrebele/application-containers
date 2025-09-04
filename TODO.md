

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
