@0xdca555fc76741dc1;
using Go = import "/go.capnp";
$Go.package("capnp_service_registry");
$Go.import("github.com/sannyschulz/mas-service-registry/capnp_service_registry");

# struct to store a sturdyref
struct SturdyRefStored {
  sturdyRefID @0 :Text;
  serviceID @1 :Text;
  payload @2 :Text;
  usersignature @3 :Text; # TODO: rename, it is the userId, use signature for the sturdyRef owner signature
  payloaDescription @4 :Text; # (optional) a payload description for the user to understand the stored service
}

# interface to add a sturyref to the registry
interface StorageEditor {

  addSturdyRef @0 (sturdyref :SturdyRefStored) -> ();
  getSturdyRef @1 (sturdyRefID :Text) -> (sturdyref :SturdyRefStored);
  listSturdyRefs @2 (usersignature :Text) -> (sturdyrefs :List(SturdyRefStored));
  deleteSturdyRef @3 (sturdyRefID :Text) -> ();
}

interface StorageReader {
  getSturdyRef @0 (sturdyRefID :Text) -> (sturdyref :SturdyRefStored) ;
}

struct UserStored {
  usersignature @0 :Text;
  sturdyRefToken @1 :Text;
}

interface UserEditor {
  newUser @0 () -> (user :UserStored);
  deleteUser @1 (usersignature :Text) -> ();
  findByToken @2 (sturdyRefToken :Text) -> (usersignature :Text);
  findBySignature @3 (usersignature :Text) -> (sturdyRefToken :Text);
  addSeal @4 (usersignature :Text, seal :Text) -> (); 
}