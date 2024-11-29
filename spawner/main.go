package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

func main() {
	configPath := flag.String("config", "", "config file")
	configGen := flag.Bool("config-gen", false, "generate a config file") // generate a config file and exit
	flag.Parse()

	// read the config file, if it exists
	var config *commonlib.Config
	var err error
	if *configGen {
		gen := &ConfigConfiguratorImpl{}
		// generate a config file if it does not exist yet
		config, err = commonlib.ConfigGen(*configPath, gen)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Config file generated at:", *configPath)
		return
	} else {
		config, err = commonlib.ReadConfig(*configPath, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	spawnConfig := getSpawnServiceConfig(config)

	sp := NewSpawnManager(spawnConfig)
	listenForRequests(sp.requestServiceMsgC, sp.registerServiceMsgC, config)

	// provide ServiceViewer and ServiceResolver capabilities

}

type ConfigConfiguratorImpl struct {
}

func (c *ConfigConfiguratorImpl) GetDefaultConfig() *commonlib.Config {
	defaultConfig := commonlib.DefaultConfig()
	defaultConfig.Data["Service"].(map[string]interface{})["Name"] = "Spawn Service"
	defaultConfig.Data["Service"].(map[string]interface{})["Id"] = "spawn_service"
	defaultConfig.Data["Service"].(map[string]interface{})["Port"] = 0 // use any free port
	defaultConfig.Data["Service"].(map[string]interface{})["Host"] = "localhost"
	defaultConfig.Data["Service"].(map[string]interface{})["Description"] = "spawn new services"
	defaultConfig.Data["Service"].(map[string]interface{})["Mode"] = "LocalWindows" // LocalWindows, LocalUnix, Slurm

	// this is an example of a service that can be spawned, should be overwritten before starting the service
	defaultConfig.Data["Spawn"].(map[string]interface{})["ClimateService1"] = map[string]interface{}{
		"Name":        "Climate Service 1",
		"Id":          "climate_service_1",
		"Description": "Climate Service 1",
		"Path":        "path/to/climate_service_1", // path to start script folder Id_Mode.ext (climate_service_1_LocalWindows.bat,  climate_service_1_LocalUnix.sh, climate_service_1_Slurm.sh)
		"IdleTimeout": 10,                          // in minutes
	}

	return defaultConfig
}

func getSpawnServiceConfig(config *commonlib.Config) map[string]map[string]interface{} {

	numServices := len(config.Data["Spawn"].(map[string]interface{}))

	serviceConfig := make(map[string]map[string]interface{}, numServices)
	for key, value := range config.Data["Spawn"].(map[string]interface{}) {
		serviceConfig[key] = value.(map[string]interface{})
	}

	return serviceConfig
}
