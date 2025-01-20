package main

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	capnp_service_registry "github.com/sannyschulz/mas-service-registry/capnp_service_registry"
	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

// implement interface ServiceViewer server
type serviceViewer struct {
	spawnManager *SpawnManager
	persistable  *commonlib.Persistable
}

func newServiceViewer(restorer *commonlib.Restorer, spawnManager *SpawnManager) *serviceViewer {
	viewer := &serviceViewer{
		spawnManager: spawnManager,
		persistable:  commonlib.NewPersistable(restorer),
	}
	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.ServiceViewer_ServerToClient(viewer))
	}
	viewer.persistable.Cap = restoreFunc
	return viewer
}

func (sp *serviceViewer) ListServices(ctx context.Context, call capnp_service_registry.ServiceViewer_listServices) error {

	list := sp.spawnManager.listServices()

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

func (sp *serviceViewer) GetServiceView(ctx context.Context, call capnp_service_registry.ServiceViewer_getServiceView) error {

	if !call.Args().HasServiceID() {
		return errors.New("no service id provided")
	}
	serviceId, err := call.Args().ServiceID()
	if err != nil {
		return err
	}
	serviceBootstrap, err := sp.spawnManager.RequestService(serviceId)
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

func (sp *serviceViewer) GetResolvableService(ctx context.Context, call capnp_service_registry.ServiceViewer_getResolvableService) error {

	if !call.Args().HasServiceID() {
		return errors.New("no service id provided")
	}
	serviceId, err := call.Args().ServiceID()
	if err != nil {
		return err
	}
	serviceBootstrap, err := sp.spawnManager.RequestService(serviceId)
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
type serviceResolver struct {
	spawnManager *SpawnManager
	persistable  *commonlib.Persistable
}

func newServiceResolver(restorer *commonlib.Restorer, spawnManager *SpawnManager) *serviceResolver {
	resolver := &serviceResolver{
		spawnManager: spawnManager,
		persistable:  commonlib.NewPersistable(restorer),
	}
	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.ServiceResolver_ServerToClient(resolver))
	}
	resolver.persistable.Cap = restoreFunc
	return resolver
}

func (sp *serviceResolver) GetLiveCapability(ctx context.Context, call capnp_service_registry.ServiceResolver_getLiveCapability) error {

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

	serviceBootstrap, err := sp.spawnManager.RequestService(serviceId)
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

// implement interface ServiceRegistry server
type serviceRegistry struct {
	spawnManager *SpawnManager
	persistable  *commonlib.Persistable
}

func newServiceRegistry(restorer *commonlib.Restorer, spawnManager *SpawnManager) *serviceRegistry {
	registry := &serviceRegistry{
		spawnManager: spawnManager,
		persistable:  commonlib.NewPersistable(restorer),
	}
	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.ServiceRegistry_ServerToClient(registry))
	}
	registry.persistable.Cap = restoreFunc
	return registry
}

func (sp *serviceRegistry) RegisterService(ctx context.Context, call capnp_service_registry.ServiceRegistry_registerService) error {

	if !call.Args().HasServiceToken() {
		return errors.New("no service token provided")
	}
	token, err := call.Args().ServiceToken()
	if err != nil {
		return err
	}
	if !call.Args().HasService() {
		return errors.New("no service provided")
	}

	clientBootstrap := call.Args().Service()

	sp.spawnManager.registerService(token, clientBootstrap)

	return nil
}
