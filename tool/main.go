package main
import _ "os"
import "fmt"
import "slices"
import "encoding/json"
import "io/ioutil"
import "maps"

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


	var storePaths = getStorePaths("firefox", "fc-cache")
	var dependencies = getRecursivePaths(slices.Collect(maps.Values(storePaths)))



	fmt.Printf("%+v\n", dependencies)
}
