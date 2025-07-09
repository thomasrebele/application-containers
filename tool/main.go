package main

import "os"
import "fmt"
import "encoding/json"
import "io/ioutil"
import "path/filepath"
import "os/user"


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

	pod := buildPodConfig(jsonConf)

	// config dir setup
	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".config", "application-containers-tool")
	yamlPath := filepath.Join(configPath, "generated")
	err = os.MkdirAll(yamlPath, os.ModePerm)
	if err != nil {
		panic(err)
	}



	outputPath := filepath.Join(yamlPath, pod.name + ".yaml")
	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(pod.yaml)
	fmt.Printf("created config file %s\n", outputPath)
	
}
