package main

import (
	"fmt"

	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

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
		"Name":         "Climate Service 1",
		"Id":           "climate_service_1",
		"Description":  "Climate Service 1",
		"Type":         "ClimateService",
		"Path":         "path/to/climate_service_1", // path to start script folder Id_Mode.ext (climate_service_1_LocalWindows.bat,  climate_service_1_LocalUnix.sh, climate_service_1_Slurm.sh)
		"IdleTimeout":  10,                          // in minutes
		"StartTimeout": 100,                         // in seconds (time to wait for the service to start)
	}

	return defaultConfig
}

func checkConfigIntegrity(config *commonlib.Config) error {
	// check if the service is configured
	if _, ok := config.Data["Service"]; !ok {
		return fmt.Errorf("no service configuration found in the config file")
	}

	// check if the spawn service is configured
	if _, ok := config.Data["Spawn"]; !ok {
		return fmt.Errorf("no spawn service configuration found in the config file")
	}
	return nil
}

func getSpawnServiceConfig(config *commonlib.Config) map[string]map[string]interface{} {

	numServices := len(config.Data["Spawn"].(map[string]interface{}))

	serviceConfig := make(map[string]map[string]interface{}, numServices)
	for key, value := range config.Data["Spawn"].(map[string]interface{}) {
		serviceConfig[key] = value.(map[string]interface{})
	}

	return serviceConfig
}
