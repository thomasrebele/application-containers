package main

import "os"
import "fmt"
import "encoding/json"
import "io/ioutil"



func main() {
	// read json config
	bs, err := ioutil.ReadFile("firefox.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	var jsonConf = map[string]interface{} {}
	err = json.Unmarshal(bs, &jsonConf)
	if err != nil {
		fmt.Println(err)
		return
	}

	output := buildPodConfig(jsonConf)

	outputPath := "/tmp/experiment.yaml"
	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(output)
	fmt.Printf("created config file %s\n", outputPath)


}
