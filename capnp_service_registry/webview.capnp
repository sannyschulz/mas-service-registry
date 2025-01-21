@0xd0ab2f9fdcbdf774;
using Go = import "/go.capnp";
$Go.package("capnp_service_registry");
$Go.import("github.com/sannyschulz/mas-service-registry/capnp_service_registry");

# struct to store a sturdyref
struct SturdyRefAdminView {
  sturdyRefID @0 :Text;
  serviceID @1 :Text;
  payload @2 :Text;
  usersignature @3 :Text;
  payloaDescription @4 :Text; # a payload description for the user to understand the service
}

struct ServiceReference {
  serviceID @0 :Text;
  serviceType @1 :Text;
  serviceName @2 :Text;
  serviceDescription @3 :Text;
  #TBD: maybe add addional information or active status
}


struct SturdyRefUserView {
  sturdyRefID @0 :Text;
  serviceID @1 :Text;
  payloaDescription @2 :Text; # a payload description for the user to understand the service
}

# interface to add a sturyref to the registry
interface WebViewUser {

    # list all services
    listServices @0 (serviceType :Text) -> (services :List(ServiceReference));
    # get a service view by id, it is up to the service to define its view 
    getServiceView @1 (serviceID :Text) -> (serviceView :Capability);
    # get a new sturdyRef as sharable ID by serviceID and specification for the user
    # payload specification should come from the service, selected by the user... check for manipulation
    newSturdyRef @2 (serviceID :Text, specification :Text) -> (sturdyRef :Text);
    # remove a sturdyRef, for the user
    removeSturdyRef @3 (sturdyRef :Text) -> ();

    # list user sturdyRefs
    listSturdyRefs @4 () -> (sturdyRefs :List(SturdyRefUserView));

}

interface WebViewAdmin {

    # list all services
    listServices @0 (serviceType :Text) -> (services :List(ServiceReference));
    # get a service view by id, it is up to the service to define its view 
    getServiceView @1 (serviceID :Text) -> (serviceView :Capability);
    # get a new sturdyRef as sharable ID by serviceID and specification
    newSturdyRef @2 (usersignature :Text, serviceID :Text, specification :Text) -> (sturdyRef :Text);
    # remove a sturdyRef
    removeSturdyRef @3 (usersignature :Text, sturdyRef :Text) -> ();

    # list user sturdyRefs
    listSturdyRefs @4 (usersignature :Text) -> (sturdyRefs :List(SturdyRefAdminView));
    # list all sturdyRefs
    listAllSturdyRefs @5 () -> (sturdyRefs :List(SturdyRefAdminView));

}