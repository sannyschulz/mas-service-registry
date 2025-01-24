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
	mgr := commonlib.NewConnectionManager(*configPath)
	// connect to storage service
	storageSturdyRef := config.Data["StorageEditor"].(string)

	// establish a connection (retry 10 times, wait 1 second between retries)
	storageCap, err := mgr.TryConnect(storageSturdyRef, 10, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer storageCap.Release()

	// connect to spawner service
	serviceViewSturdyRef := config.Data["ServiceViewer"].(string)
	// establish a connection (retry 10 times, wait 1 second between retries)
	serviceViewCap, err := mgr.TryConnect(serviceViewSturdyRef, 10, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer serviceViewCap.Release()

	// listen for requests from clients
	err = listenForRequests(config, storageCap, serviceViewCap)

	if err != nil {
		log.Fatal(err)
	}
}

func listenForRequests(config *commonlib.Config, storageCap, serviceViewCap *capnp.Client) error {

	host := config.Data["Service"].(map[string]interface{})["Host"].(string)
	port := config.Data["Service"].(map[string]interface{})["Port"].(int)
	// sturdy ref resolving part (from commonlib)
	restorer := commonlib.NewRestorer(host, uint16(port))

	// TODO: implement web service viewer for user and admin
	webViewAdmin := newWebViewAdmin(restorer, storageCap, serviceViewCap)

	webViewAdminSr, err := webViewAdmin.persistable.InitialSturdyRef()
	if err != nil {
		return err
	}
	fmt.Println("web view admin sturdy ref:", webViewAdminSr)

	// todo: need to implement the web view user restorer

	// start listening for incoming connections from clients
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
	defaultConfig.Data["Service"].(map[string]interface{})["Name"] = "Web ServiceViewer Service"
	defaultConfig.Data["Service"].(map[string]interface{})["Id"] = "web_view"
	defaultConfig.Data["Service"].(map[string]interface{})["Port"] = 0 // use any free port
	defaultConfig.Data["Service"].(map[string]interface{})["Host"] = "localhost"
	defaultConfig.Data["Service"].(map[string]interface{})["Description"] = "view and administer sturdyrefs for services"
	defaultConfig.Data["ServiceViewer"] = ""
	defaultConfig.Data["StorageEditor"] = ""

	return defaultConfig
}

func checkConfigIntegrity(config *commonlib.Config) error {
	// check if the service is configured
	if _, ok := config.Data["Service"]; !ok {
		return fmt.Errorf("no service configuration found in the config file")
	}

	// check if the sturdyrefs for storage and service viewer are configured
	if _, ok := config.Data["ServiceViewer"]; !ok {
		return fmt.Errorf("missing ServiceViewerSturdyRef in the config file")
	}
	if _, ok := config.Data["StorageEditor"]; !ok {
		return fmt.Errorf("missing StorageEditor SturdyRef in the config file")
	}

	return nil
}
