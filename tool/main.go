package main

import "os"
import "fmt"
import "encoding/json"
import "io/ioutil"
import "path/filepath"
import "os/user"
import "os/exec"
import "flag"

// TODO list
// - add a command for rebuilding the base image
// - reduce the base image to the bare minimum (no bash, coreutils, busybox, ...)
// - use a sidecar for debugging

type Act struct {
	configPath string
	yamlPath string
}

func setup() Act {
	// config dir setup
	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".config", "application-containers-tool")
	yamlPath := filepath.Join(configPath, "generated")
	err := os.MkdirAll(yamlPath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return Act{configPath, yamlPath}
}

func (act *Act) updatePodConfig(jsonPath string) (Pod, bool) {
	// read json config
	bs, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}

	var jsonConf = map[string]interface{} {}
	err = json.Unmarshal(bs, &jsonConf)
	if err != nil {
		panic(err)
	}

	pod := buildPodConfig(jsonConf)
	outputPath := filepath.Join(act.yamlPath, pod.name + ".yaml")
	pod.yamlPath = &outputPath

	// check if file exists
	if _, err = os.Stat(*pod.yamlPath); os.IsExist(err) {
		// check whether we would change its content
		bs, err = ioutil.ReadFile(*pod.yamlPath)
		if err != nil {
			panic(err)
		}
		var oldContent = string(bs)
		if oldContent == pod.yaml {
			return pod, false
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(pod.yaml)
	return pod, true
}

func runIgnoreErrors(cmd string, args ...string) {
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
		fmt.Println(string(output))
		fmt.Println(err)
	}
}

func run(cmd string, args ...string) {
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
		fmt.Println(string(output))
		panic(err)
	}
}

func main() {
	var act = setup()

	// if necessary, define global flags here
	flag.Parse()
	nextArgs := flag.Args()

	if len(nextArgs) == 0 {
		panic("Usage: [<options>] <command>")
	}

	switch nextArgs[0] {
	case "json":
		sub := flag.NewFlagSet("json", flag.ExitOnError)
		_ = run
		sub.Parse(flag.Args()[1:])
		nextArgs = sub.Args()
		if len(nextArgs) != 1 {
			panic("Specify a json file to run!")
		}

		var pod, changed = act.updatePodConfig(nextArgs[0])
		if changed {
			fmt.Println("stopping " + pod.name)
			runIgnoreErrors("podman", "pod", "stop", pod.name)
			runIgnoreErrors("podman", "pod", "rm", pod.name)
			fmt.Println("playing " + (*pod.yamlPath))
			runIgnoreErrors("podman", "play", "kube", *pod.yamlPath)

		} else {
			runIgnoreErrors("podman", "pod", "start", pod.name)
		}
		
	}

	fmt.Println("done")
}
