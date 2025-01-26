module github.com/sannyschulz/mas-service-registry/web-view

go 1.23.3

require (
	capnproto.org/go/capnp/v3 v3.0.1-alpha.2
	github.com/sannyschulz/mas-service-registry/capnp_service_registry v0.0.0-00010101000000-000000000000
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/persistence v0.0.0-20250117170510-b66cc8c23d60
	github.com/zalf-rpm/mas-infrastructure/src/go/commonlib v0.0.0-20250117170510-b66cc8c23d60
)

require (
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/common v0.0.0-20230412105359-2d45c32db41e // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/geo v0.0.0-20230208160538-deb034d36602 // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/grid v0.0.0-20230713163933-4c7223175aeb // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
)

replace github.com/sannyschulz/mas-service-registry/capnp_service_registry => ../capnp_service_registry
