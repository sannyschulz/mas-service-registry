package main

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

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

	// get the spawner service
	// fut, release = capnp_service_registry.Spawner(rh.spawnerCap)
	// if err != nil {
	// 	return capnp.ErrorClient(err), err
	// }
	// defer release()

	// TODO:need to implement the spawner service
	err = errors.New("not implemented")

	return capnp.ErrorClient(err), err
}
