package main

import (
	"os"
	"fmt"
	"os/exec"
	"strings"
	"path/filepath"
)

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

func getStorePaths(commands ...string) map[string]string {
	var result = map[string]string{}

	for _, command := range commands {
		output, err := exec.Command("which", command).Output()
		if err != nil {
			fmt.Printf("Warning: command '%s' not found: %s\n", command, err)
			continue
		}
		var path = strings.TrimSpace(string(output))
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			fmt.Printf("Warning: could not resolve path '%s' for command '%s': %s", path, command, err)
			continue
		}

		if !strings.HasPrefix(realPath, "/nix/store") {
			fmt.Printf("Warning: the path of command '%s' is not within /nix/store: %s", command, path)
			continue
		}
		result[command] = realPath
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


