package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

func main() {
	configPath := flag.String("config", "", "config file")
	configGen := flag.Bool("config-gen", false, "generate a config file")
	echo := flag.Bool("echo", false, "echo service configuration") // for debugging
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
		PrintConfig(config, *echo)
		return // exit after generating the config file
	} else {
		config, err = commonlib.ReadConfig(*configPath, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	PrintConfig(config, *echo)

	mgr := commonlib.NewConnectionManager(*configPath)
	// connect to storage service
	registrySturdyRef := config.Data["Registry"].(string)
	if registrySturdyRef == "" {
		log.Fatal("No registry sturdy ref provided")
	}
	token := config.Data["Token"].(string)
	if token == "" {
		log.Fatal("No token provided")
	}
	// establish a connection (retry 10 times, wait 1 second between retries)
	registry, err := mgr.TryConnect(registrySturdyRef, 10, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer registry.Release()

	// register with token
	err = registerWithToken(*registry, token)
	if err != nil {
		log.Fatal(err)
	}
}

func registerWithToken(registry capnp.Client, token string) error {

	// create message channels
	msgChan := make(chan string)
	errChan := make(chan error)

	// create service to spawner
	serviceToSpawner := capnp_service_registry.ServiceToSpawner_ServerToClient(NewServiceToSpawner(msgChan, errChan, true))

	// register service
	fut, release := capnp_service_registry.ServiceRegistry(registry).RegisterService(context.Background(), func(params capnp_service_registry.ServiceRegistry_registerService_Params) error {
		err := params.SetServiceToken(token)
		if err != nil {
			return err
		}
		err = params.SetService(serviceToSpawner)
		return err
	})
	defer release()
	_, err := fut.Struct()
	if err != nil {
		return err
	}

	// wait for messages
	for {
		select {
		case msg := <-msgChan:
			fmt.Println(msg)
		case err := <-errChan:
			fmt.Println(err)
		}
	}
}

type ConfigConfiguratorImpl struct {
}

func (c *ConfigConfiguratorImpl) GetDefaultConfig() *commonlib.Config {
	defaultConfig := commonlib.DefaultConfig()
	defaultConfig.Data["Token"] = ""
	defaultConfig.Data["Registry"] = ""
	defaultConfig.Data["Service"].(map[string]interface{})["Name"] = "Template Service"
	defaultConfig.Data["Service"].(map[string]interface{})["Id"] = "template_service"
	defaultConfig.Data["Service"].(map[string]interface{})["Description"] = "example for services"

	return defaultConfig
}

func PrintConfig(config *commonlib.Config, echo bool) {
	if echo {
		fmt.Println("Service configuration:")
		fmt.Println("Name:", config.Data["Service"].(map[string]interface{})["Name"])
		fmt.Println("Id:", config.Data["Service"].(map[string]interface{})["Id"])
		fmt.Println("Description:", config.Data["Service"].(map[string]interface{})["Description"])
		fmt.Println("Token:", config.Data["Token"])
		fmt.Println("Registry:", config.Data["Registry"])
	}
}
