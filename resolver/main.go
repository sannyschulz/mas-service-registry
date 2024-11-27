package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	persistence "github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/persistence"
	commonlib "github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

func main() {
	configPath := flag.String("config", "", "config file")
	configGen := flag.Bool("config-gen", false, "generate a config file")
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
	} else {
		config, err = commonlib.ReadConfig(*configPath, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	mgr := commonlib.NewConnectionManager(*configPath)
	// establish a connection to storage service
	storageSturdyRef := config.Data["Storage"].(map[string]interface{})["SturdyRef"].(string)
	if storageSturdyRef == "" {
		log.Fatal("No storage sturdy ref provided")
	}

	storageCap, err := mgr.TryConnect(storageSturdyRef, 10, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer storageCap.Release()

	// establish a connection to spawner service
	spawnerSturdyRef := config.Data["Spawner"].(map[string]interface{})["SturdyRef"].(string)
	if spawnerSturdyRef == "" {
		log.Fatal("No spawner sturdy ref provided")
	}
	spawnerCap, err := mgr.TryConnect(spawnerSturdyRef, 10, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer spawnerCap.Release()

	// listen for requests from clients
	err = listenForRequests(config, storageCap, spawnerCap)
	if err != nil {
		log.Fatal(err)
	}
}

func listenForRequests(config *commonlib.Config, storageCap, spawnerCap *capnp.Client) error {
	host := config.Data["Service"].(map[string]interface{})["Host"].(string)
	port := config.Data["Service"].(map[string]interface{})["Port"].(int)
	restorer := commonlib.NewRestorer(host, uint16(port))

	// sturdy ref resolving part

	// start listening for connections
	listener, err := config.ListenForConnections(restorer.Host(), restorer.Port())
	if err != nil {
		return err
	}
	defer listener.Close()

	errChan := make(chan error)
	msgChan := make(chan string)
	// accept incomming connection from clients
	go func() {
		main := persistence.Restorer_ServerToClient(restorer)
		defer main.Release()
		for {
			c, err := listener.Accept()
			fmt.Printf("service: request from %v\n", c.RemoteAddr())
			if err != nil {
				errChan <- err
				continue
			}
			serve(c, capnp.Client(main.AddRef()), errChan, msgChan)
		}

	}()

	for {
		select {
		case msg := <-msgChan:
			fmt.Println(msg)
		case err := <-errChan:
			fmt.Println(err)
		}
	}
}

func serve(conn net.Conn, boot capnp.Client, errChan chan error, msgChan chan string) {

	// Listen for calls, using  bootstrap interface.
	rpc.NewConn(rpc.NewStreamTransport(conn), &rpc.Options{BootstrapClient: boot, Logger: &commonlib.ConnError{Out: errChan, Msg: msgChan}})
	// this connection will be close when the client closes the connection
}

type ConfigConfiguratorImpl struct {
}

func (c *ConfigConfiguratorImpl) GetDefaultConfig() *commonlib.Config {
	defaultConfig := commonlib.DefaultConfig()
	defaultConfig.Data["Service"].(map[string]interface{})["Name"] = "Resolver Service"
	defaultConfig.Data["Service"].(map[string]interface{})["Id"] = "Resolver_service"
	defaultConfig.Data["Service"].(map[string]interface{})["Port"] = 0 // use any free port
	defaultConfig.Data["Service"].(map[string]interface{})["Host"] = "localhost"
	defaultConfig.Data["Service"].(map[string]interface{})["Description"] = "transform SturdyRefs into live capabilites"

	defaultConfig.Data["Storage"].(map[string]interface{})["SturdyRef"] = ""
	defaultConfig.Data["Spawner"].(map[string]interface{})["SturdyRef"] = ""
	return defaultConfig
}
