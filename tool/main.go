package main
import _ "os"
import "fmt"
import "encoding/json"
import "io/ioutil"

func main() {
	fmt.Println("main")

	bs, err := ioutil.ReadFile("firefox.json")
	if err != nil {
		fmt.Println(err)
		return
	}


	//jsonBlob := string(bs)
	//fmt.Println(jsonBlob)

	var conf = map[string]interface{} {}
	json.Unmarshal(bs, &conf)
	//fmt.Printf("%+v\n", conf)


	
	xyz := M(
		P{"apiVersion", "v1"},
		P{"metadata", M(
			P{"labels", M(P{"app", "firefox-pod"})},
			P{"name", "firefox-pod"},
		)},
		
	)

	xyz2 := M(
		P{"metadata", M(P{"test", "abc123"})},
		P{"spec", M(
			P{"containers", A(M(
				P{"command", "firefox"},
				P{"env", A(
					M(P{"name", "TERM"}, P{"value", "xterm"}),
					M(P{"name", "WAYLAND_DISPLAY"}, P{"value", "wayland-1"}),
				)},
			))},
		)},
	)


	xyz3 := M(
		P{"spec", M(
			P{"containers", A(M(
				P{"env", A(
					M(P{"name", "FONTS"}, P{"value", "neo-font"}),
				)},
			))},
		)},
	)


	xyz = xyz.merge(xyz2)
	xyz = xyz.merge(xyz3)

	yaml := toYaml(xyz)

	fmt.Println(yaml)

	_ = xyz2
}
