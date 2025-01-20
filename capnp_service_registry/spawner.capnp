@0xd7bbd917b8b51661;
using Go = import "/go.capnp";
$Go.package("capnp_service_registry");
$Go.import("github.com/sannyschulz/mas-service-registry/capnp_service_registry");

# struct to store a sturdyref
struct ResolvableServiceRequest {
  serviceID @0 :Text;
  payload @1 :Text;
}

# interface to resolve a service by its id and specification
interface ServiceResolver {

    # resolve a service
    getLiveCapability @0 (request :ResolvableServiceRequest) -> (resolvedCapability :Capability);

}   

struct ServiceDescription {
  serviceID @0 :Text;
  serviceType @1 :Text;
  serviceName @2 :Text;
  serviceDescription @3 :Text;
}

interface ServiceViewer {

    # list all services
    listServices @0 (serviceType :Text) -> (services :List(ServiceDescription));
    # get a service view by id, it is up to the service to define its view 
    getServiceView @1 (serviceID :Text) -> (serviceView :Capability);
    # get resolvable service reference as context of a sturdyRef, stored in storage service
    getResolvableService @2 (serviceID :Text, specification :Text) -> (service :ResolvableServiceRequest);
}

interface ServiceRegistry {

    # register a service
    registerService @0 (serviceToken :Text, service :Capability) -> ();

}

interface ServiceToSpawner {

    # resolve payload to capability
    getLiveCapability @0 (payload :Text) -> (resolvedCapability :Capability);
    # get a service view, it is up to the service to define its view 
    getServiceView @1 () -> (serviceView :Capability);
    # get a specification to a service and receive a payload, with which it can be resolved
    getResolvablePayload @2 (specification :Text) -> (payload :Text);

}