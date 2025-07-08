package main
import _ "os"
import "fmt"
import "slices"
import "encoding/json"
import "io/ioutil"
import "maps"

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
				SP("env", A(
				)),
			))),
		)),
	)

	baseConfig := M(
		P("spec", M(
			P("containers", A(M(
				P("env", A(
					M(P("name", "TERM"), P("value", "xterm")),
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
			P("containers", A(M(
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
			P("containers", A(M(
				P("name", fmt.Sprintf("%s%s",name,"-container")),
			))),
		)),
	)



	// home config
	containerHomePath := fmt.Sprintf("/home/%s", name)
	targetHomePath := conf["home"].(map[string]interface{})["path"]
	homeConfig := M(
		P("spec", M(
			P("containers", A(M(
				P("env", A(
					M(P("name", "HOME"), P("value", containerHomePath)),
				)),
				P("volumeMounts", A(M(
					P("mountPath", containerHomePath),
					P("name", "home-dir"),
				))),
			))),
			P("volumes", A(M(
				P("name", "home-dir"),
				P("hostPath", M(
					P("path", targetHomePath),
					P("type", "Directory"),
				)),
			))),
		)),
	)

	

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
		merge(homeConfig)
	output := toYaml(yaml)
	fmt.Println(output)

}
