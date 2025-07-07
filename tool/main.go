package main
import _ "os"
import "fmt"
import "reflect"
import "strings"
import "encoding/json"
import "io/ioutil"

type KeyValue struct {
    Key   string
    Value interface{}
}

type P = KeyValue

type Map struct {
	items []KeyValue
}

func M(keyValues ...KeyValue) Map {
	return Map{items: keyValues};
}

func A(items ...Map) []Map {
	return items;
}

type OutputState int;
const (
	OutputStateListStart OutputState = iota
	OutputStateMap 
	OutputStateValue
)

func toYaml(data interface{}) string {
	var sb strings.Builder
	toYamlHelper(data, &sb, 0, OutputStateMap)
	return sb.String()
}

func toYamlHelper(data interface{}, sb *strings.Builder,
		indent int, cont OutputState) {
	prefix := strings.Repeat("  ", indent)
	switch v := data.(type) {
	case string:
		sb.WriteString(fmt.Sprintf(" %s", v))

	case Map:
		first := true
		for _, item := range v.items {
			if !first || (cont != OutputStateMap && cont != OutputStateListStart) {
				sb.WriteString("\n")
			}
			if !first || cont != OutputStateListStart {
				sb.WriteString(fmt.Sprintf("%s", prefix))
			}
			first = false

			sb.WriteString(item.Key)
			sb.WriteString(":")
			toYamlHelper(item.Value, sb, indent+1, OutputStateValue)
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
				toYamlHelper(item, sb, indent, OutputStateListStart)
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


	
	xyz := M(
		P{"apiVersion", "v1"},
		P{"metadata", M(
			P{"labels", M(P{"app", "firefox-pod"})},
			P{"name", "firefox-pod"},
		)},
		
	)

	xyz2 := M(
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

	// TODO
	//xyz = xyz.merge(xyz2)

	yaml := toYaml(xyz)

	fmt.Println(yaml)

	_ = xyz2
}
