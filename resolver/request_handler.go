package main

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

// interface for capability forwarding handler (from commonlib)
type resolveHandler struct {
	storageCap capnp.Client
	spawnerCap capnp.Client
}

// CanResolveSturdyRef checks if the SturdyRefToken exists in the storage
func (rh *resolveHandler) CanResolveSturdyRef(srToken string) bool {

	// check if sturdyRef exists in storage
	fut, release := capnp_service_registry.StorageReader(rh.storageCap).GetSturdyRef(context.Background(), func(params capnp_service_registry.StorageReader_getSturdyRef_Params) error {
		err := params.SetSturdyRefID(srToken)
		return err
	})
	defer release()
	futStruct, err := fut.Struct()
	if err != nil {
		return false
	}
	if futStruct.HasSturdyref() {
		return true
	}
	return false
}

// ResolveSturdyRef resolves a SturdyRefToken to a capability
func (rh *resolveHandler) ResolveSturdyRef(srToken string) (capnp.Client, error) {
	// if it exists, generating a capability from the sturdyRef may still fail

	// get the sturdyRef from storage
	fut, release := capnp_service_registry.StorageReader(rh.storageCap).GetSturdyRef(context.Background(), func(params capnp_service_registry.StorageReader_getSturdyRef_Params) error {
		err := params.SetSturdyRefID(srToken)
		return err
	})
	defer release()
	futStruct, err := fut.Struct()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	if !futStruct.HasSturdyref() {
		// sturdyRef not found in storage
		err := errors.New("SturdyRef not found")
		return capnp.ErrorClient(err), err
	}
	// get info for spawner service
	stStored, err := futStruct.Sturdyref()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	stStored.HasServiceID()
	// get the serviceID
	serviceID, err := stStored.ServiceID()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	// payload for spawner service
	payload, err := stStored.Payload()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	fmt.Print("ServiceID: ", serviceID)
	fmt.Print("Payload: ", payload)

	//get the spawner service and resolve the capability
	futResolve, releaseResolve := capnp_service_registry.ServiceResolver(rh.spawnerCap).GetLiveCapability(context.Background(),
		func(params capnp_service_registry.ServiceResolver_getLiveCapability_Params) error {

			// create a request struct
			resquest, err := params.NewRequest()
			if err != nil {
				return err
			}
			err = resquest.SetServiceID(serviceID)
			if err != nil {
				return err
			}
			err = resquest.SetPayload(payload)
			if err != nil {
				return err
			}

			return params.SetRequest(resquest)
		})

	defer releaseResolve()

	futResolveStruct, err := futResolve.Struct()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	if !futResolveStruct.HasResolvedCapability() {
		err := errors.New("capability cannot be resolved")
		return capnp.ErrorClient(err), err
	}
	resolvedCap := futResolveStruct.ResolvedCapability().AddRef()

	return resolvedCap, nil
}
