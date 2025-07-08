package main
import "os"
import "fmt"
import "slices"
import "encoding/json"
import "io/ioutil"
import "maps"


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

func addVolume(conf Map, vol Volume) Map {
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
					P("name", vol.name),
				))),
			))),
			P("volumes", A(M(
				P("name", vol.name),
				P("hostPath", M(
					P("path", vol.hostPath),
					P("type", filetype),
				)),
			))),
		)),
	))
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

func main() {
	fmt.Println("main")

	// read json config
	bs, err := ioutil.ReadFile("firefox.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	var conf = map[string]interface{} {}
	err = json.Unmarshal(bs, &conf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(fmt.Sprintf("test: %s\n",conf))

	var storePaths = getStorePaths("firefox", "fc-cache")
	var dependencies = getRecursivePaths(slices.Collect(maps.Values(storePaths)))
	//fmt.Printf("%+v\n", dependencies)
	_ = dependencies

	// default pod options
	skeleton := M(
		P("apiVersion", "v1"),
		P("metadata", M()),
		P("spec", M(
			SP("containers", A(M(
				P("name", nil),
				P("image", nil),
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

	// TODO check which properties can be removed
	name := conf["name"]
	namingConfig := M(
		P("metadata", M(
			P("name", name),
			P("labels", M(
				P("app", name),
			)),
		)),
		P("spec", M(
			SP("containers", A(M(
				P("name", fmt.Sprintf("%s%s",name,"-container")),
			))),
		)),
	)

	// home config
	containerHomePath := fmt.Sprintf("/home/%s", name)
	targetHomePath := conf["home"].(map[string]interface{})["path"].(string)
	homeConfig := envConfig(env("HOME", containerHomePath))
	homeConfig = addVolume(homeConfig,
		mount(targetHomePath).
		to(containerHomePath).
		as("home-dir"))

	// config provided by features
	featureConfig := M()
	var runPath = fmt.Sprintf("/run/user/%d",uid)
	for _, f := range conf["features"].([]interface{}) {
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
			fconf = addVolume(fconf, mount(runPath + "/wayland-1").as("f-wayland"))

		case "pulse":
			fconf = envConfig(
				env("PULSE_SERVER", "unix:/run/user/1001/pulse/native"))
			fconf = addVolume(fconf, mount(runPath + "/pulse").as("f-pulse"))

		case "fonts":
			fconf = envConfig(
				env("FONTCONFIG_PATH", "/etc/fonts"))
			fconf = addVolume(fconf, mount("/etc/fonts").as("f-fonts"))

		case "cacert":
			// TODO avoid hardcoding!
			fconf = M()
			fconf = addVolume(fconf, mount("/etc/ssl").as("f-cacert-1"))
			fconf = addVolume(fconf, mount("/etc/static/ssl").as("f-cacert-2"))
			fconf = addVolume(fconf, mount("/nix/store/b9anbghrppj43ci27fh0zyawis1plxik-nss-cacert-3.111/etc/ssl/certs/ca-bundle.crt").as("f-cacert-3"))

		case "webcam":
			var count = 0
			fconf = M()
			for _, device := range options["devices"].([]interface{}) {
				d := device.(string)
				fmt.Println(d)
				fconf = addVolume(fconf, mount("/dev/" + d).as("f-webcam-" + d))
				count += 1
			}

		default:
			fmt.Printf("Warning: option %s not yet supported\n", name)
		}
		featureConfig = featureConfig.merge(fconf)
	}


//	xyz3 := M(
//		P("metadata", M(
//			P("labels", M(P("app", "firefox-pod"))),
//			P("name", "firefox-pod"),
//		)),
//		P("spec", M(
//			P("containers", A(M(
//				P("env", A(
//					M(P("name", "FONTS"), P("value", "neo-font")),
//				)),
//			))),
//		)),
//	)

	yaml := skeleton.
		merge(namingConfig).
		merge(securityConfig).
		merge(baseConfig).
		merge(homeConfig).
		merge(featureConfig)
	output := toYaml(yaml)
	fmt.Println(output)

}
