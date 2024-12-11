package main

import (
	"context"
	"errors"

	capnp_service_registry "github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

// implement interface ServiceViewer server

func (sp *SpawnManager) ListServices(ctx context.Context, call capnp_service_registry.ServiceViewer_listServices) error {

	list := sp.listServices()

	result, err := call.AllocResults()
	if err != nil {
		return err
	}
	structList, err := result.NewServices(int32(len(list)))
	if err != nil {
		return err
	}

	for i, service := range list {
		serviceDesc, err := capnp_service_registry.NewServiceDescription(structList.Segment())
		if err != nil {
			return err
		}
		err = serviceDesc.SetServiceID(service.Id)
		if err != nil {
			return err
		}
		err = serviceDesc.SetServiceName(service.Name)
		if err != nil {
			return err
		}
		err = serviceDesc.SetServiceDescription(service.Description)
		if err != nil {
			return err
		}
		err = serviceDesc.SetServiceType(service.Type)
		if err != nil {
			return err
		}

		structList.Set(i, serviceDesc)
	}
	return result.SetServices(structList)
}

func (sp *SpawnManager) GetServiceView(ctx context.Context, call capnp_service_registry.ServiceViewer_getServiceView) error {

	if !call.Args().HasServiceID() {
		return errors.New("no service id provided")
	}
	serviceId, err := call.Args().ServiceID()
	if err != nil {
		return err
	}
	serviceBootstrap, err := sp.RequestService(serviceId)
	if err != nil {
		return err
	}
	service := capnp_service_registry.ServiceToSpawner(serviceBootstrap)
	fut, release := service.GetServiceView(ctx, func(sr capnp_service_registry.ServiceToSpawner_getServiceView_Params) error {
		return nil
	})
	defer release()
	liveCap, err := fut.Struct()
	if err != nil {
		return err
	}

	result, err := call.AllocResults()
	if err != nil {
		return err
	}
	return result.SetServiceView(liveCap.ServiceView().AddRef())
}

func (sp *SpawnManager) GetResolvableService(ctx context.Context, call capnp_service_registry.ServiceViewer_getResolvableService) error {

	if !call.Args().HasServiceID() {
		return errors.New("no service id provided")
	}
	serviceId, err := call.Args().ServiceID()
	if err != nil {
		return err
	}
	serviceBootstrap, err := sp.RequestService(serviceId)
	if err != nil {
		return err
	}
	specs, err := call.Args().Specification()
	if err != nil {
		return err
	}
	if len(specs) == 0 {
		return errors.New("no specification provided")
	}

	service := capnp_service_registry.ServiceToSpawner(serviceBootstrap)
	fut, release := service.GetResolvablePayload(ctx, func(sr capnp_service_registry.ServiceToSpawner_getResolvablePayload_Params) error {
		return sr.SetSpecification(specs)
	})
	defer release()
	resolvablePayload, err := fut.Struct()
	if err != nil {
		return err
	}
	payload, err := resolvablePayload.Payload()
	if err != nil {
		return err
	}

	result, err := call.AllocResults()
	if err != nil {
		return err
	}

	serviceAnswer, err := result.NewService()
	if err != nil {
		return err
	}
	err = serviceAnswer.SetPayload(payload)
	if err != nil {
		return err
	}
	err = serviceAnswer.SetServiceID(serviceId)
	if err != nil {
		return err
	}
	return result.SetService(serviceAnswer)
}

// implement interface ServiceResolver server

func (sp *SpawnManager) GetLiveCapability(ctx context.Context, call capnp_service_registry.ServiceResolver_getLiveCapability) error {

	req, err := call.Args().Request()
	if err != nil {
		return err
	}
	if !req.HasServiceID() {
		return errors.New("no service id provided")
	}

	serviceId, err := req.ServiceID()
	if err != nil {
		return err
	}
	payload, err := req.Payload()
	if err != nil {
		return err
	}

	serviceBootstrap, err := sp.RequestService(serviceId)
	if err != nil {
		return err
	}
	resolver := capnp_service_registry.ServiceToSpawner(serviceBootstrap)
	// resolve the payload by using the bootstrap capability of the service

	fut, release := resolver.GetLiveCapability(ctx, func(sr capnp_service_registry.ServiceToSpawner_getLiveCapability_Params) error {
		return sr.SetPayload(payload)
	})
	defer release()
	liveCap, err := fut.Struct()
	if err != nil {
		return err
	}

	result, err := call.AllocResults()
	if err != nil {
		return err
	}

	return result.SetResolvedCapability(liveCap.ResolvedCapability().AddRef())
}
