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

func (map1 *Map) index() map[string]KeyValue {
	kv := map[string]KeyValue {};
	for _, item := range map1.items {
		kv[item.Key] = item
	}
	return kv;
}

func (map1 Map) merge(map2 Map) Map {
	var result Map
	map2index := map2.index()

	var done = map[string]bool{}

	// iterate over map1 and merge with values from map2 if available
	for _, item := range map1.items {
		var k1 = item.Key
		var v1 = item.Value

		var v2t, exists = map2index[k1]
		if !exists {
			result.items = append(result.items, KeyValue{k1, v1})
			done[k1] = true
			continue
		}
		var v2 = v2t.Value

		switch v := v1.(type) {
			case Map:
				if reflect.TypeOf(v2) != reflect.TypeOf(v1) {
					panic(fmt.Sprintf("corresponding type of attribute '%s' needs to be a map, but was %s", k1, reflect.TypeOf(v2)))
				}

				var merged = v1.(Map).merge(v2.(Map))
				result.items = append(result.items, KeyValue{k1, merged})
				done[k1] = true
				continue

			case []Map:
				if reflect.TypeOf(v2) != reflect.TypeOf(v1) {
					panic(fmt.Sprintf("corresponding type of attribute '%s' needs to be an array, but was %s", k1, reflect.TypeOf(v2)))
				}
				var merged = []Map{}
				merged = append(merged, v1.([]Map)...)
				merged = append(merged, v2.([]Map)...)
				result.items = append(result.items, KeyValue{k1, merged})
				done[k1] = true
				continue
			
			default:
				panic(fmt.Sprintf("Unknown type: %s", reflect.TypeOf(v)))
		}
		panic(fmt.Sprintf("values not mergeable: %s and %s", reflect.TypeOf(v1), reflect.TypeOf(v2)))
	}

	// add remaining entries from map2
	for _, item := range map2.items {
		var k2 = item.Key
		var _, exists = done[k2]
		if exists {
			continue
		}

		var v2 = item.Value
		result.items = append(result.items, KeyValue{k2, v2})
	}

	return result;
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
		P{"metadata", M(P{"test", "abc"})},
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
