package main

import "reflect"
import "fmt"
import "strings"

// An example how this experimental code could be used:

//	xyz := M(
//		P{"apiVersion", "v1"},
//		P{"metadata", M(
//			P{"labels", M(P{"app", "firefox-pod"})},
//			P{"name", "firefox-pod"},
//		)},
//		
//	)
//
//	xyz2 := M(
//		P{"metadata", M(P{"test", "abc123"})},
//		P{"spec", M(
//			P{"containers", A(M(
//				P{"command", "firefox"},
//				P{"env", A(
//					M(P{"name", "TERM"}, P{"value", "xterm"}),
//					M(P{"name", "WAYLAND_DISPLAY"}, P{"value", "wayland-1"}),
//				)},
//			))},
//		)},
//	)
//
//
//	xyz3 := M(
//		P{"spec", M(
//			P{"containers", A(M(
//				P{"env", A(
//					M(P{"name", "FONTS"}, P{"value", "neo-font"}),
//				)},
//			))},
//		)},
//	)
//
//	xyz = xyz.merge(xyz2)
//	xyz = xyz.merge(xyz3)
//	yaml := toYaml(xyz)
//	fmt.Println(yaml)

type OptArrayMerge int;
const (
	OptArrayMergeConcat = iota
	// merge by copying the content of the sole (!) item of the second array
	// into each items of the first
	OptArrayMergeIntegrate
)


type KeyValue struct {
    Key   string
    Value interface{}
	ArrayMergeMode OptArrayMerge
}

type Map struct {
	items []KeyValue
}

// 'M' stands for "map"
func M(keyValues ...KeyValue) Map {
	return Map{items: keyValues};
}

// 'A' stands for "array"
func A(items ...Map) []Map {
	return items;
}

// 'P' stands for "property"
func P(key string, value interface{}) KeyValue {
	return KeyValue{key, value, OptArrayMergeConcat}
}

// 'P' stands for "singular property"
// i.e., its value is a single-element array;
// merging such an array will result in a single-element array
func SP(key string, value interface{}) KeyValue {
	return KeyValue{key, value, OptArrayMergeIntegrate}
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
			result.items = append(result.items, P(k1, v1))
			done[k1] = true
			continue
		}
		var v2 = v2t.Value
		// always overwrite nil values
		// (nil values may be used for ordering the properties)
		if v1 == nil {
			result.items = append(result.items, P(k1, v2))
			done[k1] = true
			continue
		}

		switch v1.(type) {
			case Map:
				if reflect.TypeOf(v2) != reflect.TypeOf(v1) {
					panic(fmt.Sprintf("corresponding type of property '%s' needs to be a map, but was %s", k1, reflect.TypeOf(v2)))
				}

				var merged = v1.(Map).merge(v2.(Map))
				result.items = append(result.items, P(k1, merged))
				done[k1] = true
				continue

			case []Map:
				if reflect.TypeOf(v2) != reflect.TypeOf(v1) {
					panic(fmt.Sprintf("corresponding type of property '%s' needs to be an array, but was %s", k1, reflect.TypeOf(v2)))
				}
				var merged = []Map{}
				if item.ArrayMergeMode == OptArrayMergeConcat {
					merged = append(merged, v1.([]Map)...)
					merged = append(merged, v2.([]Map)...)
				} else if item.ArrayMergeMode == OptArrayMergeIntegrate {
					var v2m []Map = v2.([]Map)
					if len(v2m) != 1 {
						panic(fmt.Sprintf("the second array of property '%s' must have one element, but its length was %d: %+v", k1, len(v2m), v2m))
					}
					var other = v2m[0]
					
					for _, v := range v1.([]Map) {
						merged = append(merged, v.merge(other))
					}
					if k1 == "env" {
						fmt.Printf("p %s v1 %s v2 %s merged %s\n", k1, v1, v2, merged)
					}
				} else {
					panic("Unknown array merge option")
				}
				result.items = append(result.items, KeyValue{k1, merged, item.ArrayMergeMode})
				done[k1] = true
				continue
			
		}
		panic(fmt.Sprintf("values not mergeable for property '%s': %s and %s", k1, reflect.TypeOf(v1), reflect.TypeOf(v2)))
	}

	// add remaining entries from map2
	for _, item := range map2.items {
		var k2 = item.Key
		var _, exists = done[k2]
		if exists {
			continue
		}

		var v2 = item.Value
		result.items = append(result.items, P(k2, v2))
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
	case int:
		sb.WriteString(fmt.Sprintf(" %d", v))
	case bool:
		if v == true {
			sb.WriteString(" true")
		} else {
			sb.WriteString(" false")
		}

	case Map:
		first := true
		for _, item := range v.items {
			if item.Value != nil {
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


