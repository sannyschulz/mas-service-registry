package main

import (
	"context"
	"errors"

	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

// implement interface ServiceToSpawner
// # resolve payload to capability
// getLiveCapability @0 (payload :Text) -> (resolvedCapability :Capability);
// # get a service view, it is up to the service to define its view
// getServiceView @1 (callback :SaveCallback) -> (serviceView :Capability);
//

type serviceToSpawner struct {
	msgChan  chan string
	errChan  chan error
	debugOut bool
}

func NewServiceToSpawner(msgChan chan string, errChan chan error, debugOut bool) *serviceToSpawner {
	return &serviceToSpawner{
		msgChan:  msgChan,
		errChan:  errChan,
		debugOut: debugOut,
	}
}

func (s *serviceToSpawner) GetLiveCapability(ctx context.Context, call capnp_service_registry.ServiceToSpawner_getLiveCapability) error {
	// read the payload
	payload, err := call.Args().Payload()
	if err != nil {
		// write error to the error channel
		s.errChan <- err
		return err
	}
	if s.debugOut {
		s.msgChan <- "GetLiveCapability: " + payload
	}
	// return the capability
	// TODO: implement the capability
	return errors.New("not implemented")
}

func (s *serviceToSpawner) GetServiceView(ctx context.Context, call capnp_service_registry.ServiceToSpawner_getServiceView) error {

	return nil
}
