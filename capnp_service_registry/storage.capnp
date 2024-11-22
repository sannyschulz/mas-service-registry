@0xdca555fc76741dc1;
using Go = import "/go.capnp";
$Go.package("capnp_service_registry");
$Go.import("github.com/sannyschulz/mas-service-registry/capnp_service_registry");

# struct to store a sturdyref
struct SturdyRefStored {
  sturdyRefID @0 :Text;
  serviceID @1 :Text;
  payload @2 :Text;
  usersignature @3 :Text;
}

# interface to add a sturyref to the registry
interface StorageEditor {

  addSturdyRef @0 (sturdyref :SturdyRefStored) -> ();
  getSturdyRef @1 (sturdyRefID :Text) -> (sturdyref :SturdyRefStored);
  listSturdyRefsForUser @2 (usersignature :Text) -> (sturdyrefs :List(SturdyRefStored));
  listAllSturdyRefs @3 () -> (sturdyrefs :List(SturdyRefStored));
  deleteSturdyRef @4 (sturdyRefID :Text) -> ();
}

interface StorageReader {
  getSturdyRef @0 (sturdyRefID :Text) -> (sturdyref :SturdyRefStored) ;
  # maybe later add more methods to read data
}