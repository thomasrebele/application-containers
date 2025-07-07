package main
import _ "os"
import "fmt"
import "reflect"
import "strings"
import "encoding/json"
import "io/ioutil"

type Pair struct {
    Key   string
    Value interface{}
}

type K Pair;

type Map struct {
	items []K
}

func M(keyValues ...K) Map {
	return Map{items: keyValues};
}

func A(items ...Map) []Map {
	return items;
}

type YamlNext int;
const (
	YamlNextListStart YamlNext = iota
	YamlNextKeyValue 
	YamlNextValue
)

func toYaml(data interface{}) string {
	var sb strings.Builder
	toYamlHelper(data, &sb, 0, YamlNextKeyValue)
	return sb.String()
}

func toYamlHelper(data interface{}, sb *strings.Builder,
		indent int, cont YamlNext) {
	prefix := strings.Repeat("  ", indent)
	switch v := data.(type) {
	case string:
		sb.WriteString(fmt.Sprintf(" %s", v))

	case Map:
		first := true
		for _, value := range v.items {
			if !first || (cont != YamlNextKeyValue && cont != YamlNextListStart) {
				sb.WriteString("\n")
			}
			if !first || cont != YamlNextListStart {
				sb.WriteString(fmt.Sprintf("%s", prefix))
			}
			first = false
			toYamlHelper(value, sb, indent, YamlNextValue)
		}

	case K:
		first := true
		if !first || cont != YamlNextListStart {
			sb.WriteString(fmt.Sprintf("%s", prefix))
		}
		sb.WriteString(v.Key)
		sb.WriteString(":")
		toYamlHelper(v.Value, sb, indent+1, YamlNextValue)

	case map[string]interface{}:
		first := true
		for key, value := range v {
			if !first || (cont != YamlNextKeyValue && cont != YamlNextListStart) {
				sb.WriteString("\n")
			}
			if !first || cont != YamlNextListStart {
				sb.WriteString(fmt.Sprintf("%s", prefix))
			}
			first = false
			sb.WriteString(fmt.Sprintf("%s:", key))
			toYamlHelper(value, sb, indent+1, YamlNextValue)
		}
	default:
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Slice {
			prefix := strings.Repeat("  ", indent-1)
			for i:=0; i<rv.Len(); i++ {
				if sb.Len() != 0 {
					sb.WriteString("\n")
				}
				item := rv.Index(i).Interface()
				sb.WriteString(fmt.Sprintf("%s- ", prefix))
				toYamlHelper(item, sb, indent, YamlNextListStart)
			}
			return;
		}

		sb.WriteString(fmt.Sprintf("unknown type: %+v", v))
	}
}

//func printYaml(data []KeyValue) string {
//	var sb strings.Builder
//
//	sb.WriteString("test")
//	for _, str := range data {
//		sb.WriteString(str.Key())
//	}
//
//	return sb.String()
//}


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

	var output = map[string]interface{} {
		"apiVersion": "v1",
		"metadata": map[string]interface{} {
			"labels": map[string]interface{} {
				"app": "firefox-pod",
			},
			"name": "firefox-pod",
		},
		//"spec": map[string]interface{} {
		//	"containers": []string{
		//		"",
		//	}.
		//},
		"spec": map[string]interface{} {
			"containers": []map[string]interface{} {
				{
					"command": "firefox",
					"env": []map[string]interface{} {
						{
							"name": "TERM",
							"value": "xterm",
						},
						{
							"name": "WAYLAND_DISPLAY",
							"value": "wayland-1",
						},

					},
				},
			},
		},
	}
	yaml := toYaml(output)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(yaml)
	fmt.Println("--------------------------------------------------------------------------------")

	

	

	//xyz := []Pair {
	//	{"apiVersion", "v1"},
	//	{"metadata", []Pair{
	//		{"labels", []Pair{
	//			{"xyz", "123"},
	//		}},
	//	}},
	//}

	xyz := M(
		K{"apiVersion", "v1"},
		K{"metadata", M(
			K{"labels", M(K{"app", "firefox-pod"})},
			K{"name", "firefox-pod"},
		)},
		K{"spec", M(
			K{"containers", A(M(
				K{"command", "firefox"},
				K{"env", A(
					M(K{"name", "TERM"}, K{"value", "xterm"}),
					M(K{"name", "WAYLAND_DISPLAY"}, K{"value", "wayland-1"}),
				)},
			))},
		)},
	)
	yaml = toYaml(xyz)
	fmt.Println(yaml)

}
