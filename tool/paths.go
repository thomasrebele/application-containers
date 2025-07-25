package main

import (
	"os"
	"syscall"
	"fmt"
	"os/exec"
	"strings"
	"path/filepath"
)

func escapeShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func runWithStatus(command string, args ...string) int {
	cmd := exec.Command(command, args...)
	_, err := cmd.Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
}

func getCommandPath(command string) *string {
	output, err := exec.Command("which", command).Output()
	if err != nil {
		fmt.Printf("Warning: command '%s' not found: %s\n", command, err)
		return nil
	}
	var path = strings.TrimSpace(string(output))
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		fmt.Printf("Warning: could not resolve path '%s' for command '%s': %s", path, command, err)
		return nil
	}
	return &realPath
}

func getCommandPaths(commands ...string) map[string]string {
	var result = map[string]string{}

	for _, command := range commands {
		var path = getCommandPath(command)
		if path != nil {
			result[command] = *path
		}
	}

	return result
}

func resolveMainCommand(name string) *string {
	var result = getCommandPath(name)
	if result != nil {
		return result;
	}

	var path = getStorePathForPackageName(name);
	var result2 = *path + "/bin/" + name;
	return &result2;
}

func getStorePathForPackageName(name string) *string {
	expr := fmt.Sprintf("(import <nixpkgs> {}).%s.outPath", name)
	_ = expr;
	output, err := exec.Command("nix-instantiate", "--eval-only", "--expr", expr).Output()
	if err != nil {
		fmt.Printf("Warning: could not get store path for package %s:XXX %s\n", name, err)
		return nil
	}

	var path = strings.TrimSpace(string(output))
	path = path[1:len(path)-1]
	return &path
}

func getStorePathsForCommands(commands ...string) map[string]string {
	var result = map[string]string{}

	for _, command := range commands {
		var path = getCommandPath(command);
		if path == nil {
			path = getStorePathForPackageName(command);
			if path == nil {
				fmt.Printf("Warning: store path for command '%s' could not be resolved\n", command)
				continue
			}
		}

		if !strings.HasPrefix(*path, "/nix/store") {
			fmt.Printf("Warning: the path of command '%s' is not within /nix/store: %s\n", command, path)
			continue
		}
		result[command] = *path
	}

	return result;
}

func getDependeeStorePaths(paths []string) map[string]bool {
	var result = map[string]bool{}
	for _, path := range paths {
		collectDependeeStorePaths(path, &result)
	}
	return result;
}

func collectDependeeStorePaths(path string, result *map[string]bool) {
	output, err := exec.Command("nix-store", "-qR", path).Output()
	if err != nil {
		fmt.Printf("Warning: could not get path because of error: %s\n", err)
		return
	}

	for _, p := range strings.Split(string(output), "\n") {
		if p == "" {
			return
		}
		(*result)[p] = true
	}
}

func getDependeePaths(path string) map[string]bool {
	var result = map[string]bool{}
	collectDependeePaths(path, &result);
	return result
}

func collectDependeePaths(path string, result *map[string]bool) {
	info, err := os.Lstat(path)
	if err != nil {
		panic(fmt.Sprintf("failed to stat %s: %w", path, err))
	}

	// nix store paths have their own dependees
	if strings.HasPrefix(path, "/nix/store/") {
		(*result)[path] = true
		collectDependeeStorePaths(path, result)
		return
	}

	// decend into symbolic links
	if info.Mode() & os.ModeSymlink != 0 {
		(*result)[path] = true
		target, err := os.Readlink(path)
		if err != nil {
			fmt.Println("Error reading symlink:", err)
			return
		}
		collectDependeePaths(target, result)
		return
	}

	// decend into directories
	if !info.IsDir() {
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory %s: %w", path, err))
	}

	for _, entry := range entries {

		fullPath := filepath.Join(path, entry.Name())
		collectDependeePaths(fullPath, result)
	}

}


