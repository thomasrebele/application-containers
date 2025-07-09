package main

import (
	"fmt"
	"os/exec"
	"strings"
	"path/filepath"
)


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

	fmt.Println(result);

	return result;
}

func getDependeeStorePaths(paths []string) map[string]bool {
	var result = map[string]bool{}

	for _, path := range paths {
		output, err := exec.Command("nix-store", "-qR", path).Output()
		if err != nil {
			fmt.Printf("Warning: could not get path because of error: %s\n", err)
			continue
		}

		for _, p := range strings.Split(string(output), "\n") {
			if p == "" {
				continue
			}
			result[p] = true
		}
	}

	return result;
}

func getFontStorePaths() map[string]bool {
	var result = map[string]bool{}

	output, err := exec.Command("fc-list", "-f%{file}\n").Output()
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "/nix/store/") {
			continue
		}
		var parts = strings.Split(line, "/")
		var l = strings.Join(parts[:4], "/")
		result[string(l)] = true
	}


	return result
}
