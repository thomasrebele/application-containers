package main

import "os"
import "fmt"
import "slices"
import "maps"
import "path/filepath"

type Pod struct {
	name string
	yaml string
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

	info, err := os.Stat(vol.hostPath)
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
	entries, err := os.ReadDir(containerPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory %s: %w", containerPath, err))
	}

	for _, entry := range entries {
		fullPath := filepath.Join(containerPath, entry.Name())

		info, err := os.Lstat(fullPath)
		if err != nil {
			panic(fmt.Sprintf("failed to stat %s: %w", fullPath, err))
		}

		if info.Mode() & os.ModeSymlink != 0 {
			(*volumes)[fullPath] = fullPath
		} 
		if info.IsDir() {
			addVolumeRecursively(volumes, fullPath)
		}
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

	// home config
	containerHomePath := fmt.Sprintf("/home/%s", name)
	targetHomePath := jsonConf["home"].(map[string]interface{})["path"].(string)
	homeConfig := envConfig(env("HOME", containerHomePath))
	volumes[containerHomePath] = targetHomePath

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
				env("XDG_RUNTIME_DIR", runPath))
			volumes[runPath + "/wayland-1"] = runPath + "/wayland-1"

		case "pulse":
			fconf = envConfig(
				env("PULSE_SERVER", "unix:/run/user/1001/pulse/native"))
			volumes[runPath + "/pulse"] = runPath + "/pulse"

		case "fonts":
			// TODO avoid hardcoding
			fconf = envConfig(
				env("FONTCONFIG_PATH", "/etc/fonts"))
			volumes["/etc/fonts"] = "/etc/fonts"
			volumes["/etc/static/fonts"] = "/etc/static/fonts"
			addVolumeRecursively(&volumes, "/etc/fonts")

			for font, _ := range getFontStorePaths() {
				fmt.Println(font)
				volumes[font] = font
			}



		case "cacert":
			fconf = M()
			volumes["/etc/ssl"] = "/etc/ssl"
			volumes["/etc/static/ssl"] = "/etc/static/ssl"
			// TODO avoid hardcoding!
			// use nix-instantiate --eval-only --expr '(import <nixpkgs> {}).cacert.outPath' instead
			cacertPath := "/nix/store/b9anbghrppj43ci27fh0zyawis1plxik-nss-cacert-3.111/etc/ssl/certs/ca-bundle.crt"
			volumes[cacertPath] = cacertPath

		case "webcam":
			fconf = M()
			for _, device := range options["devices"].([]interface{}) {
				d := device.(string)
				volumes["/dev/" + d] = "/dev/" + d
			}

		default:
			fmt.Printf("Warning: option %s not yet supported\n", name)
		}
		featureConfig = featureConfig.merge(fconf)
	}

	
	// TODO use nix-instantiate --eval-only --expr '(import <nixpkgs> {}).cacert.outPath' instead
	var packages1 = jsonConf["packages"].([]interface{})
	var packages = make([]string, len(packages1))
	for i, v := range packages1 {
		packages[i] = v.(string)
	}
	var storePaths = getStorePaths(packages...)
	var dependencies = getDependeeStorePaths(slices.Collect(maps.Values(storePaths)))
	//fmt.Printf("%+v\n", dependencies)
	_ = dependencies

	for dep, _ := range dependencies {
		volumes[dep] = dep
	}

	// TODO setup the main command!
	commandConfig := M(
		P("spec", M(
			SP("containers", A(M(
				P("command", []string{"/bin/start-script.sh"}),
				P("args", []string{"/nix/store/xzx3l4kh8dvngvlhfzsn7936klwvd4mv-firefox-139.0.1/bin/firefox"}),
			))),
		)),
	)

	// TODO avoid copying the map again and again!
	storeConfig := M()
	for containerPath, hostPath := range volumes {
		storeConfig = addVolumeConfig(storeConfig, mount(hostPath).to(containerPath))
	}

	yaml := skeleton.
		merge(namingConfig).
		merge(commandConfig).
		merge(securityConfig).
		merge(baseConfig).
		merge(homeConfig).
		merge(featureConfig).
		merge(storeConfig)
	return Pod{name, toYaml(yaml)}
}
