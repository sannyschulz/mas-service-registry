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
		_, err = commonlib.ConfigGen(*configPath, gen)
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
	err = checkConfigIntegrity(config)
	if err != nil {
		log.Fatal(err)
	}
	spawnConfig := getSpawnServiceConfig(config)
	listenForRequests(NewSpawnManager(spawnConfig), config)

	// provide ServiceViewer and ServiceResolver capabilities

}

func listenForRequests(spM *SpawnManager, config *commonlib.Config) {

}
