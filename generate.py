import os
import subprocess
import sys
import yaml

def get_store_paths(commands):
    """Get the Nix store paths for each command, resolving symlinks and extracting the binary path."""
    store_paths = {}
    for cmd in commands:
        try:
            symlink_path = subprocess.check_output(["which", cmd]).decode().strip()
            real_path = os.path.realpath(symlink_path)  # Resolve symlink

            if real_path.startswith("/nix/store"):
                store_paths[cmd] = real_path
        except subprocess.CalledProcessError:
            print(f"Warning: {cmd} not found in PATH", file=sys.stderr)

    return store_paths

def generate_yaml(template_file, commands, command):
    """Generate the YAML configuration with volume mounts and command execution."""
    store_paths = get_store_paths(commands)

    dependencies = set()
    for path in store_paths.values():
        dependencies.update(subprocess.check_output(["nix-store", "-qR", path]).decode().splitlines())

    # Read the existing template YAML configuration
    with open(template_file, "r") as f:
        config = yaml.safe_load(f)

    # Ensure necessary keys exist
    if "spec" not in config:
        config["spec"] = {}

    if "containers" not in config["spec"] or not config["spec"]["containers"]:
        raise ValueError("No containers defined in the template file")

    # Get the first container
    container = config["spec"]["containers"][0]

    # Ensure volumeMounts key exists
    if "volumeMounts" not in container:
        container["volumeMounts"] = []

    # Ensure command and args keys exist
    if "command" not in container:
        container["command"] = []

    # Append new volume mounts
    for dep in dependencies:
        mount_name = f"nix-store-{dep.split('/')[-1]}"
        container["volumeMounts"].append({"name": mount_name, "mountPath": dep})

    # Ensure volumes exist at the top level
    if "volumes" not in config["spec"]:
        config["spec"]["volumes"] = []

    # Append volume definitions
    for dep in dependencies:
        mount_name = f"nix-store-{dep.split('/')[-1]}"
        config["spec"]["volumes"].append({
            "name": mount_name,
            "hostPath": {
                "path": dep,
                "type": "Directory"
            }
        })

    # Set the container's startup command using resolved paths
    container["command"] = ["/bin/start-script.sh"]
    container["args"] = [store_paths[command[0]]] + command[1:]

    return yaml.dump(config)

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <template.yaml> [--add command2]... <command> [arg] ...", file=sys.stderr)
        sys.exit(1)

    template_file = sys.argv[1]

    commands = []
    i = 1
    while i < len(sys.argv)-1:
        i+=1
        if sys.argv[i] == "--add":
            i+=1
            commands += [sys.argv[i]]
        else:
            break

    commands += [sys.argv[i]]
    command = sys.argv[i:]

    print(generate_yaml(template_file, commands, command))

