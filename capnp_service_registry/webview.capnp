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
    # remove a sturdyRef, for the user
    removeSturdyRef @2 (sturdyRef :Text) -> ();
    # list user sturdyRefs
    listSturdyRefs @3 () -> (sturdyRefs :List(SturdyRefUserView));
}

interface WebViewAdmin {

    # remove a sturdyRef from user
    removeSturdyRef @0 (usersignature :Text, sturdyRef :Text) -> ();
    # list all sturdyRefs 
    # if userFilter is empty, list all
    listAllSturdyRefs @1 (userFilter :List(Text)) -> (sturdyRefs :List(SturdyRefAdminView));

    # get a new user for the webview, with a signature, if the user usersignature is empty, it will be created
    # the Idea is that the admin will create the capability for the user
    # if the user already exists, it should return the stored WebViewUser Capability
    newWebViewUser @2 (usersignature :Text) -> (webViewUser :WebViewUser);
}

