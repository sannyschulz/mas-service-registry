module github.com/sannyschulz/mas-service-registry/resolver

go 1.23.3

require (
	capnproto.org/go/capnp/v3 v3.0.1-alpha.2
	github.com/sannyschulz/mas-service-registry/capnp_service_registry v0.0.0-20241127170230-c8a42a68846e
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/persistence v0.0.0-20241128114125-73a4e2e1843b
	github.com/zalf-rpm/mas-infrastructure/src/go/commonlib v0.0.0-20241128114125-73a4e2e1843b
)

require (
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/common v0.0.0-20241128114125-73a4e2e1843b // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/geo v0.0.0-20241128114125-73a4e2e1843b // indirect
	github.com/zalf-rpm/mas-infrastructure/capnproto_schemas/gen/go/grid v0.0.0-20241128114125-73a4e2e1843b // indirect
	golang.org/x/crypto v0.29.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
)

replace github.com/sannyschulz/mas-service-registry/capnp_service_registry => ../capnp_service_registry