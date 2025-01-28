package main

import (
	"context"
	"fmt"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	persistence "github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/persistence"
	commonlib "github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"

	capnp_service_registry "github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

// listen for requests from clients
func listenForRequests(request chan *dbRequest, config *commonlib.Config) error {
	// accept incoming connections
	host := config.Data["Service"].(map[string]interface{})["Host"].(string)
	port := config.Data["Service"].(map[string]interface{})["Port"].(int)
	restorer := commonlib.NewRestorer(host, uint16(port)) // port 0 means: use any free port

	// start listening for connections
	listener, err := config.ListenForConnections(restorer.Host(), restorer.Port())
	if err != nil {
		return err
	}
	defer listener.Close()

	storeEd := newStorageEditor(restorer, request)
	storeRead := newStorageReader(restorer, request)
	initialSturdyRef, err := storeEd.initialSturdyRef()
	if err != nil {
		return err
	}
	fmt.Printf("StorageEditor: %v\n", initialSturdyRef)
	initialSturdyRef, err = storeRead.initialSturdyRef()
	if err != nil {
		return err
	}
	fmt.Printf("StorageReader: %v\n", initialSturdyRef)

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

//-------------------------------------------------------------------------

// implement StorageEditor and StorageReader interfaces

// create a new storageEditor
func newStorageEditor(restorer *commonlib.Restorer, request chan *dbRequest) *storageEditor {
	storage := &storageEditor{
		dbRequest:   request,
		persistable: commonlib.NewPersistable(restorer),
	}
	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.StorageEditor_ServerToClient(storage))
	}
	storage.persistable.Cap = restoreFunc
	return storage
}

// create a new storageReader
func newStorageReader(restorer *commonlib.Restorer, request chan *dbRequest) *storageReader {
	storage := &storageReader{
		dbRequest:   request,
		persistable: commonlib.NewPersistable(restorer),
	}
	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.StorageReader_ServerToClient(storage))
	}
	storage.persistable.Cap = restoreFunc
	return storage
}

// get the initial sturdy reference of the storageEditor
func (rs *storageReader) initialSturdyRef() (*commonlib.SturdyRef, error) {

	return rs.persistable.InitialSturdyRef()
}

// get the initial sturdy reference of the storageReader
func (es *storageEditor) initialSturdyRef() (*commonlib.SturdyRef, error) {

	return es.persistable.InitialSturdyRef()
}

// implement the StorageEditor and StorageReader interfaces
type storageEditor struct {
	dbRequest   chan *dbRequest
	persistable *commonlib.Persistable
}

func (s *storageEditor) AddSturdyRef(ctx context.Context, call capnp_service_registry.StorageEditor_addSturdyRef) error {

	sref, err := call.Args().Sturdyref()
	if err != nil {
		return err
	}
	sturdyRefId, err := sref.SturdyRefID()
	if err != nil {
		return err
	}
	payload, err := sref.Payload()
	if err != nil {
		return err
	}
	seriveId, err := sref.ServiceID()
	if err != nil {
		return err
	}
	authToken, err := sref.Usersignature()
	if err != nil {
		return err
	}
	// TODO: check if input is valid

	request := &dbRequest{
		requestType:  addSturdyRefRequest,
		sturdyRef:    sturdyRefId,
		serviceId:    seriveId,
		payload:      payload,
		authToken:    authToken,
		responseChan: make(chan dbResponse),
	}
	s.dbRequest <- request

	response := <-request.responseChan

	if response.err != nil {
		return response.err
	}

	return nil
}
func (s *storageEditor) GetSturdyRef(ctx context.Context, call capnp_service_registry.StorageEditor_getSturdyRef) error {

	sturdyRefId, err := call.Args().SturdyRefID()
	if err != nil {
		return err
	}
	if len(sturdyRefId) == 0 {
		return fmt.Errorf("sturdyRefId is empty")
	}
	request := &dbRequest{
		requestType:  getSturdyRefRequest,
		sturdyRef:    sturdyRefId,
		responseChan: make(chan dbResponse),
	}
	s.dbRequest <- request
	response := <-request.responseChan
	if response.err != nil {
		return response.err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	sref, err := results.NewSturdyref()
	if err != nil {
		return err
	}
	err = sref.SetSturdyRefID(response.sturdyRefs[0].sturdyRef)
	if err != nil {
		return err
	}
	err = sref.SetServiceID(response.sturdyRefs[0].serviceId)
	if err != nil {
		return err
	}
	err = sref.SetPayload(response.sturdyRefs[0].payload)
	if err != nil {
		return err
	}
	err = sref.SetUsersignature(response.sturdyRefs[0].authToken)
	if err != nil {
		return err
	}

	return nil
}
func (s *storageEditor) ListSturdyRefs(ctx context.Context, call capnp_service_registry.StorageEditor_listSturdyRefs) error {

	var userSignature string

	request := &dbRequest{
		responseChan: make(chan dbResponse),
	}
	var err error
	if call.Args().HasUsersignature() {

		userSignature, err = call.Args().Usersignature()
		if err != nil {
			return err
		}
		request.requestType = listSturdyRefsByAuthTokenRequest
		request.authToken = userSignature
	} else {
		request.requestType = listSturdyRefsRequest
	}
	s.dbRequest <- request
	response := <-request.responseChan
	if response.err != nil {
		return response.err
	}
	result, err := call.AllocResults()
	if err != nil {
		return err
	}
	// return if no sturdyRefs are found
	if len(response.sturdyRefs) == 0 {
		return nil
	}

	sturdyRefs, err := result.NewSturdyrefs(int32(len(response.sturdyRefs)))
	if err != nil {
		return err
	}
	for i, sref := range response.sturdyRefs {
		sturdyRefStored, err := capnp_service_registry.NewSturdyRefStored(sturdyRefs.Segment())
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetPayload(sref.payload)
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetServiceID(sref.serviceId)
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetSturdyRefID(sref.sturdyRef)
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetUsersignature(sref.authToken)
		if err != nil {
			return err
		}
		err = sturdyRefs.Set(i, sturdyRefStored)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *storageEditor) DeleteSturdyRef(ctx context.Context, call capnp_service_registry.StorageEditor_deleteSturdyRef) error {

	sturdyRefId, err := call.Args().SturdyRefID()
	if err != nil {
		return err
	}
	if len(sturdyRefId) == 0 {
		return fmt.Errorf("sturdyRefId is empty")
	}
	request := &dbRequest{
		requestType:  deleteSturdyRefRequest,
		sturdyRef:    sturdyRefId,
		responseChan: make(chan dbResponse),
	}
	s.dbRequest <- request
	response := <-request.responseChan
	if response.err != nil {
		return response.err
	}

	return nil
}

type storageReader struct {
	dbRequest   chan *dbRequest
	persistable *commonlib.Persistable
}

func (s *storageReader) GetSturdyRef(ctx context.Context, call capnp_service_registry.StorageReader_getSturdyRef) error {
	return nil
}
