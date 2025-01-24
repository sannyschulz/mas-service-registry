@0xd7bbd917b8b51661;
using Go = import "/go.capnp";
$Go.package("capnp_service_registry");
$Go.import("github.com/sannyschulz/mas-service-registry/capnp_service_registry");

# struct for content assigned to a sturdyref
struct ResolvableServiceRequest {
  serviceID @0 :Text;
  payload @1 :Text;
}

struct ServiceDescription {
  serviceID @0 :Text;
  serviceType @1 :Text;
  serviceName @2 :Text;
  serviceDescription @3 :Text;
}

# interface to resolve a service by its id and specification to a capability
interface ServiceResolver {
    # resolve a service
    getLiveCapability @0 (request :ResolvableServiceRequest) -> (resolvedCapability :Capability);
}   

# interface to view services that are available in the spawner
interface ServiceViewer {

    # list all services
    listServices @0 (serviceType :Text) -> (services :List(ServiceDescription));
    # get a service view by id, it is up to the service to define its view 
    getServiceView @1 (serviceID :Text, callback :SaveCallback) -> (serviceView :Capability);
}

# interface to register a service at the spawner
interface ServiceRegistry {
    # register a service
    registerService @0 (serviceToken :Text, service :ServiceToSpawner) -> ();
}

# this interface has to be implemented by a service to be spawned
interface ServiceToSpawner {
    # resolve payload to capability
    getLiveCapability @0 (payload :Text) -> (resolvedCapability :Capability);
    # get a service view, it is up to the service to define its view 
    getServiceView @1 (callback :SaveCallback) -> (serviceView :Capability);
}

# callback when a live capability is saved to the storage
interface SaveCallback {
    # the payload describes a way the live capability can be restored in this specific service
    # save shall store the payload in the storage service
    save @0 (payload :Text) -> ();
}
