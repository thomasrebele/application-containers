package main

import "os"
import "fmt"
import "slices"
import "maps"
import "sort"
import "strings"
import "path/filepath"

type Pod struct {
	name string
	yaml string
	yamlPath *string
}


func env(name string, value string) Map {
	return M(P("name", name), P("value", value))
}

func envConfig(envVars ...Map) Map {
	return M(
		P("spec", M(
			SP("containers", A(M(
				P("env", A(envVars...)),
			))),
		)),
	)
}

type Volume struct {
	hostPath string
	mountPath string
	name string
}

func addVolumeConfig(conf Map, vol Volume) Map {
	var filetype = "Directory"

	info, err := os.Lstat(vol.hostPath)
	if err != nil {
		panic(err)
	}
	if !info.IsDir() {
		filetype = "File"
	}

	return conf.merge(M(
		P("spec", M(
			SP("containers", A(M(
				P("volumeMounts", A(M(
					P("mountPath", vol.mountPath),
					P("name", vol.mountPath),
				))),
			))),
			P("volumes", A(M(
				P("name", vol.mountPath),
				P("hostPath", M(
					P("path", vol.hostPath),
					P("type", filetype),
				)),
			))),
		)),
	))
}

func addVolumeRecursively(volumes *map[string]string, containerPath string) {
	var depPaths = getDependeePaths(containerPath)
	for path, _ := range depPaths {
		(*volumes)[path] = path
	}
}

func mount(hostPath string) Volume {
	return Volume{hostPath, hostPath, hostPath}
}

func (vol Volume) to(mountPath string) Volume {
	return Volume{vol.hostPath, mountPath, vol.name}
}

func (vol Volume) as(name string) Volume {
	return Volume{vol.hostPath, vol.mountPath, name}
}

func splitPath(path string) []string {
	if path == "/" {
		return []string{}
	}
	dir, last := filepath.Split(path)
	if dir == "" {
		return []string{last}
	}
	return append(splitPath(filepath.Clean(dir)), last)
}

func resolveFHSEnvSymlink(volumes *map[string]string, hostPath string, hostPathRoot string) (*string, bool) {
	fmt.Printf("resolving %s\n", hostPath)
	info, err := os.Lstat(hostPath)
	if err != nil {
		return nil, false
	}

	fmt.Printf("%s\n", info.Mode())
	if info.Mode() & os.ModeSymlink == 0 {
		return &hostPath, true
	}

	dst, err := os.Readlink(hostPath)
	fmt.Printf("dst %s\n", dst)
	if strings.HasPrefix(dst, "/nix/store/") {
		// Q&D
		parts := splitPath(dst)
		fmt.Println("%s\n", parts)
		needed := filepath.Join("/nix/store/", parts[2])
		addVolumeRecursively(volumes, needed)

		return &dst, true
	}
	if strings.HasPrefix(dst, "/proc/") {
		return nil, true
	}

	result, _ := resolveFHSEnvSymlink(volumes, filepath.Join(hostPathRoot, dst), hostPathRoot)
	if result != nil {
		return result, true
	}

	// the file does not exist at the root, so ignore it
	return nil, true
}

func integrateFHSEnvIntoMount(volumes *map[string]string, containerPath string, hostPath string, hostPathRoot string) {
	fmt.Printf("\nintegrating %s into %s\n", hostPath, containerPath)
	// check if path exists
	var containerPathExists = false
	if _, ok := (*volumes)[containerPath]; ok {
		containerPathExists = true
	}

	// check the docker image as well
	if !containerPathExists {
		path := escapeShell(containerPath)
		exists := runWithStatus("podman", "run", "--rm", "-it", "localhost/thinbase", "sh", "-c", "[[ -e " + path + " ]]")
		containerPathExists = exists == 0
	}

	info, err := os.Lstat(hostPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to stat %s: %w", hostPath, err))
	}

	fmt.Printf("%s\n", info.Mode())
	if info.Mode() & os.ModeSymlink != 0 {
		// TODO deal with rootfs symbolic links!
		resolvedPath, ok := resolveFHSEnvSymlink(volumes, hostPath, hostPathRoot)
		if resolvedPath == nil {
			if ok {return}
			panic(fmt.Sprintf("Could not resolve path %s\n", hostPath))
		}
		if strings.HasPrefix(*resolvedPath, "/nix/store") {
			if strings.Contains(*resolvedPath, "/proc") {panic("HERE")}
			(*volumes)[containerPath] = *resolvedPath
			return
		}
		if strings.HasPrefix(*resolvedPath, "/proc") {
			return
		}
		if strings.HasSuffix(*resolvedPath, "/usr/lib32") {
			// TODO: root/lib32 points to /usr/lib32, but root/usr/lib32 does not exist!
			// (seems to be a problem of buildFHSEnv)
			return
		}
		fmt.Printf("got resolved path %s\n", *resolvedPath)
		integrateFHSEnvIntoMount(volumes, containerPath, *resolvedPath, hostPathRoot)
		return
	}

	if !containerPathExists {
		(*volumes)[containerPath] = hostPath
		// Q&D we do not need the hostPath itself, but its symlink dependencies
		addVolumeRecursively(volumes, hostPath)
		return
	}

	if !info.IsDir() {
		(*volumes)[containerPath] = hostPath
		return
	}

	// decend into directories
	fmt.Printf("descend into %s\n", hostPath)
	entries, err := os.ReadDir(hostPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory %s: %w", hostPath, err))
	}

	for _, entry := range entries {
		childContainerPath := filepath.Join(containerPath, entry.Name())
		childHostPath := filepath.Join(hostPath, entry.Name())
		integrateFHSEnvIntoMount(volumes, childContainerPath, childHostPath, hostPathRoot)
	}
}

func buildPodConfig(jsonConf map[string]interface{}) Pod {
	// TODO add comment to yaml mentioning the source of the json

	// default pod options
	skeleton := M(
		P("apiVersion", "v1"),
		P("kind", "Pod"),
		P("metadata", M()),
		P("spec", M(
			SP("containers", A(M(
				P("name", nil),
				P("image", nil),
				P("command", nil),
				P("args", nil),
				P("securityContext", nil),
				P("env", A()),
				P("volumeMounts", A()),
			))),
			P("volumes", A()),
		)),
	)

	baseConfig := M(
		P("spec", M(
			SP("containers", A(M(
				P("env", A(
					env("TERM", "xterm"),
				)),
				P("image", "localhost/thinbase:latest"),
				P("stdin", true),
				P("tty", true),
			))),
			P("restartPolicy", "Never"),
		)),
	)

	var uid = 1001

	// options related to security
	securityConfig := M(
		P("metadata", M(
			P("annotations", M(P("io.podman.annotations.userns", "keep-id"))),
		)),
		P("spec", M(
			SP("containers", A(M(
				P("securityContext", M(
					P("allowPrivilegeEscalation", false),
					P("runAsUser", uid),
					P("runAsGroup", uid),
					P("fsUser", uid),
					P("fsGroup", uid),
				)),
			))),
		)),
	)

	name := "act-" + jsonConf["name"].(string)
	namingConfig := M(
		P("metadata", M(
			P("name", name),
			P("labels", M(
				P("created-by", "application-containers-tool"),
			)),
		)),
		P("spec", M(
			SP("containers", A(M(
				P("name", fmt.Sprintf("%s%s",name,"-container")),
			))),
		)),
	)

	//
	var volumes = map[string]string{}
	// use time zone of host by default
	volumes["/etc/localtime"] = "/etc/localtime"

	// home config
	containerHomePath := fmt.Sprintf("/home/%s", name)
	targetHomePath := jsonConf["home"].(string)
	homeConfig := envConfig(env("HOME", containerHomePath))
	volumes[containerHomePath] = targetHomePath

	// mount config
	if jsonConf["mount"] != nil {
		for containerPath, opts := range jsonConf["mount"].(map[string]interface{}) {
			// if there are no options, just mount at the same path
			_ = opts
			targetPath := &containerPath
			hostPath := containerPath
			switch opts.(type) {
			case map[string]interface{}:
				options := (opts.(map[string]interface{}))
				if options["hostPath"] != nil {
					hostPath = options["hostPath"].(string)
				}
				if options["integrateFHSEnv"] != nil {
					targetPath = nil
					hostPath = options["integrateFHSEnv"].(string)
					integrateFHSEnvIntoMount(&volumes, containerPath, hostPath, hostPath)
				}
			}
			if targetPath != nil {
				volumes[*targetPath] = hostPath
			}
		}
	}

	// config provided by features
	featureConfig := M()
	var runPath = fmt.Sprintf("/run/user/%d",uid)
	for _, f := range jsonConf["features"].([]interface{}) {
		var name string
		var options map[string]interface{}
		switch f.(type) {
		case string: name = f.(string)
		case map[string]interface{}: 
			options = (f.(map[string]interface{}))
			name = options["name"].(string)
		}
		
		fconf := M()
		switch name {
		case "wayland":
			fconf = envConfig(
				env("WAYLAND_DISPLAY", "wayland-1"),
				// TODO fix the owner/permissions of the runpath volume
				env("XDG_RUNTIME_DIR", runPath))
			volumes[runPath + "/wayland-1"] = runPath + "/wayland-1"

		case "pulse":
			fconf = envConfig(
				env("PULSE_SERVER", "unix:/run/user/1001/pulse/native"))
			volumes[runPath + "/pulse"] = runPath + "/pulse"

		case "fonts":
			fconf = envConfig(
				env("FONTCONFIG_PATH", "/etc/fonts"))
			addVolumeRecursively(&volumes, "/etc/fonts")

		case "cacert":
			addVolumeRecursively(&volumes, "/etc/ssl")

		case "webcam":
			for _, device := range options["devices"].([]interface{}) {
				d := device.(string)
				volumes["/dev/" + d] = "/dev/" + d
			}

		// TODO add feature "dbus"
		// dbus is necessary for 'firefox ...' commands to open a new tab
		// - /etc/dbus-1/session.conf needs to disable apparmor for certain paths?!?
		//   (see meld ../addendum/etc/dbus-1/ /etc/dbus-1)
		// - need to start dbus and set DBUS_SESSION_BUS_ADDRESS:
		//   DBUS_SESSION_BUS_ADDRESS=`/bin/dbus-daemon --fork --config-file=/etc/dbus-1/session.conf --print-address`

		default:
			fmt.Printf("Warning: option %s not yet supported\n", name)
		}
		featureConfig = featureConfig.merge(fconf)
	}

	// add /nix/store paths for commands
	var commands1 = jsonConf["commands"].([]interface{})
	var commands = make([]string, len(commands1))
	for i, v := range commands1 {
		commands[i] = v.(string)
	}
	var storePaths = getStorePathsForCommands(commands...)
	var dependencies = getDependeeStorePaths(slices.Collect(maps.Values(storePaths)))
	for dep, _ := range dependencies {
		volumes[dep] = dep
	}

	// add /nix/store paths for packages
	if jsonConf["packages"] != nil {
		var packages1 = jsonConf["packages"].([]interface{})
		var packages = make([]string, len(packages1))
		for i, v := range packages1 {
			packages[i] = v.(string)
		}
		for _, path := range packages {
			storePath := getStorePathForPackageName(path)
			dependencies = getDependeeStorePaths([]string{*storePath})
			for dep, _ := range dependencies {
				volumes[dep] = dep
			}
		}
	}

	// start-script is used to setup dbus
	// dbus is necessary for 'firefox ...' commands to open a new tab

	// TODO setup the main command!
	//commandConfig := M(
	//	P("spec", M(
	//		SP("containers", A(M(
	//			P("command", []string{"/bin/start-script.sh"}),
	//			P("args", []string{"/nix/store/xzx3l4kh8dvngvlhfzsn7936klwvd4mv-firefox-139.0.1/bin/firefox"}),
	//		))),
	//	)),
	//)

	var command *string;

	runOption := jsonConf["run"]

	if runOption == nil {
		command = resolveMainCommand(commands[0])
	} else {
		runCommand := runOption.(string)
		command = &runCommand
	}
	if command == nil {
		panic("Error: could not infer the main command")
	}

	commandConfig := M(
		P("spec", M(
			SP("containers", A(M(
				P("command", []string{"/bin/sh"}),
				P("args", []string{"-c", "\"export DBUS_SESSION_BUS_ADDRESS=`/bin/dbus-daemon --fork --config-file=/etc/dbus-1/session.conf --print-address`; " + *command + "\""}),
			))),
		)),
	)


	// TODO avoid copying the map again and again!
	storeConfig := M()
	keys := make([]string, 0, len(volumes))
	for containerPath, _ := range volumes {
		keys = append(keys,containerPath)
	}
	sort.Strings(keys)
	for _, key := range keys {
		storeConfig = addVolumeConfig(storeConfig, mount(volumes[key]).to(key))
	}

	yaml := skeleton.
		merge(namingConfig).
		merge(commandConfig).
		merge(securityConfig).
		merge(baseConfig).
		merge(homeConfig).
		merge(featureConfig).
		merge(storeConfig)
	return Pod{name, toYaml(yaml), nil}
}
