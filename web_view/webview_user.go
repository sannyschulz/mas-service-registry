package main

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
)

// implement interface WebViewUser
type webViewUser struct {
	storedCap   capnp.Client
	serviceView capnp.Client
	userID      string
}

func newWebViewUser(storecap, viewerCap capnp.Client, userId string) *webViewUser {
	wv := &webViewUser{
		storedCap:   storecap,
		serviceView: viewerCap,
		userID:      userId,
	}
	return wv
}

func webViewMessageLoop() {

	for {
		// do something
		// handle store cap callbacks
		// handle
	}
}

// type WebViewUser_Server interface {
func (wv *webViewUser) ListServices(ctx context.Context, call capnp_service_registry.WebViewUser_listServices) error {
	return nil
}

func (wv *webViewUser) GetServiceView(ctx context.Context, call capnp_service_registry.WebViewUser_getServiceView) error {
	return nil
}

func (wv *webViewUser) RemoveSturdyRef(ctx context.Context, call capnp_service_registry.WebViewUser_removeSturdyRef) error {
	return nil
}

func (wv *webViewUser) ListSturdyRefs(ctx context.Context, call capnp_service_registry.WebViewUser_listSturdyRefs) error {
	return nil
}

// interface for capability forwarding handler (from commonlib)
type restoreWebViewHandler struct {
	userEditor capnp.Client
	storedcap  capnp.Client
	viewerCap  capnp.Client

	loadedCaps map[string]*webViewUser
	getCap     chan getCap
}

func newRestoreWebViewHandler(storecap, viewerCap, userEditor capnp.Client) *restoreWebViewHandler {
	handler := &restoreWebViewHandler{
		userEditor: userEditor,
		storedcap:  storecap,
		viewerCap:  viewerCap,
		loadedCaps: make(map[string]*webViewUser),
		getCap:     make(chan getCap),
	}

	// handle sturdyRef resolving
	go func() {
		for {
			select {
			case get := <-handler.getCap:
				if _, ok := handler.loadedCaps[get.user]; !ok {
					// create new webview user capability
					wvU := newWebViewUser(handler.storedcap, handler.viewerCap, get.user)
					handler.loadedCaps[get.user] = wvU
				}
				cap := capnp.Client(capnp_service_registry.WebViewUser_ServerToClient(handler.loadedCaps[get.user]))
				get.answer <- cap
			}
		}
	}()
	return handler
}

// CanResolveSturdyRef checks if the SturdyRefToken exists in the storage
func (rh *restoreWebViewHandler) CanResolveSturdyRef(srToken string) bool {

	// check if sturdyRef exists in storage
	fut, release := capnp_service_registry.UserEditor(rh.userEditor).FindByToken(context.Background(), func(params capnp_service_registry.UserEditor_findByToken_Params) error {
		err := params.SetSturdyRefToken(srToken)
		return err
	})
	defer release()
	futStruct, err := fut.Struct()
	if err != nil {
		return false
	}
	if futStruct.HasUsersignature() {
		return true
	}
	return false
}

// ResolveSturdyRef resolves a SturdyRefToken to a capability
func (rh *restoreWebViewHandler) ResolveSturdyRef(srToken string) (capnp.Client, error) {
	// if it exists, generating a capability from the sturdyRef may still fail

	// get the sturdyRef from storage
	fut, release := capnp_service_registry.UserEditor(rh.userEditor).FindByToken(context.Background(), func(params capnp_service_registry.UserEditor_findByToken_Params) error {
		err := params.SetSturdyRefToken(srToken)
		return err
	})
	defer release()
	futStruct, err := fut.Struct()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	if !futStruct.HasUsersignature() {
		// sturdyRef not found in storage
		err := errors.New("SturdyRef not found")
		return capnp.ErrorClient(err), err
	}
	// get info for spawner service
	userID, err := futStruct.Usersignature()
	if err != nil {
		return capnp.ErrorClient(err), err
	}
	fmt.Println("sturdyRef found in storage for user:", userID)
	// TODO: make new capability to server as WebViewUser
	capRestore := getCap{user: userID}
	rh.getCap <- capRestore
	cap := <-capRestore.answer

	return cap, err
}

type getCap struct {
	user   string
	answer chan capnp.Client
}
