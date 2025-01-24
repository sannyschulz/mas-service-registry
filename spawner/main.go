package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/persistence"
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
	// provide ServiceViewer and ServiceResolver capabilities
	err = listenForRequests(NewSpawnManager(spawnConfig), config)
	if err != nil {
		log.Fatal(err)
	}
}

func listenForRequests(spM *SpawnManager, config *commonlib.Config) error {

	host := config.Data["Service"].(map[string]interface{})["Host"].(string)
	port := config.Data["Service"].(map[string]interface{})["Port"].(int)
	restorer := commonlib.NewRestorer(host, uint16(port))

	// listen for requests for these interfaces
	// ServiceResolver
	resolver := newServiceResolver(restorer, spM)
	// ServiceViewer
	viewer := newServiceViewer(restorer, spM)
	// ServiceRegistry
	registry := newServiceRegistry(restorer, spM)

	// write the initial sturdy refs to a file
	outSturdyRefFile := config.Data["Service"].(map[string]interface{})["OutSturdyRefFile"].(string)
	err := writeSturdyRefToFile(outSturdyRefFile, resolver, viewer, registry)
	if err != nil {
		return err
	}

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

func writeSturdyRefToFile(outSturdyRefFile string, resolver *serviceResolver, viewer *serviceViewer, registry *serviceRegistry) error {
	resolverSturdyRef, err := resolver.persistable.InitialSturdyRef()
	if err != nil {
		return err
	}
	viewerSturdyRef, err := viewer.persistable.InitialSturdyRef()
	if err != nil {
		return err
	}
	registrySturdyRef, err := registry.persistable.InitialSturdyRef()
	if err != nil {
		return err
	}

	// write toml file with the sturdy refs
	defaultConfig := commonlib.DefaultConfig()
	defaultConfig.Data["Service"].(map[string]interface{})["ResolverSturdyRef"] = resolverSturdyRef
	defaultConfig.Data["Service"].(map[string]interface{})["ViewerSturdyRef"] = viewerSturdyRef
	defaultConfig.Data["Service"].(map[string]interface{})["RegistrySturdyRef"] = registrySturdyRef

	return defaultConfig.WriteConfig(outSturdyRefFile)
}
