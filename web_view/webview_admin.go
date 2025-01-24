package main

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

// implement interface WebViewUser
type webViewAdmin struct {
	storedCap   *capnp.Client
	serviceView *capnp.Client
	userEditor  *capnp.Client
	persistable *commonlib.Persistable
}

func newWebViewAdmin(restorer *commonlib.Restorer, storecap, viewerCap *capnp.Client) *webViewAdmin {
	wv := &webViewAdmin{
		persistable: commonlib.NewPersistable(restorer),
		storedCap:   storecap,
		serviceView: viewerCap,
	}

	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.WebViewAdmin_ServerToClient(wv))
	}
	wv.persistable.Cap = restoreFunc
	return wv
}

// type WebViewAdmin_Server interface
func (wv *webViewAdmin) ListServices(ctx context.Context, call capnp_service_registry.WebViewAdmin_listServices) error {
	return nil
}

func (wv *webViewAdmin) GetServiceView(ctx context.Context, call capnp_service_registry.WebViewAdmin_getServiceView) error {
	return nil
}

func (wv *webViewAdmin) NewSturdyRef(ctx context.Context, call capnp_service_registry.WebViewAdmin_newSturdyRef) error {
	return nil
}

func (wv *webViewAdmin) RemoveSturdyRef(ctx context.Context, call capnp_service_registry.WebViewAdmin_removeSturdyRef) error {
	return nil
}

func (wv *webViewAdmin) ListAllSturdyRefs(ctx context.Context, call capnp_service_registry.WebViewAdmin_listAllSturdyRefs) error {
	return nil
}

func (wv *webViewAdmin) NewWebViewUser(ctx context.Context, call capnp_service_registry.WebViewAdmin_newWebViewUser) error {
	return nil
}
