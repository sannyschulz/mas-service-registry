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
	storageEditor    capnp_service_registry.StorageEditor
	serviceView      capnp_service_registry.ServiceViewer
	userID           string
	loadedStrudyRefs []string
	storeMsgChan     chan storeMsg
}

func newWebViewUser(storeEditorCap, viewerCap capnp.Client, userId string) *webViewUser {
	wv := &webViewUser{
		storageEditor:    capnp_service_registry.StorageEditor(storeEditorCap),
		serviceView:      capnp_service_registry.ServiceViewer(viewerCap),
		userID:           userId,
		loadedStrudyRefs: []string{},
	}
	go wv.webViewMessageLoop()
	return wv
}

func (wv *webViewUser) webViewMessageLoop() {

	for {
		// handle store cap callbacks
		select {
		case storeMsg := <-wv.storeMsgChan:
			wv.StoreSturdyRef(storeMsg)
		}
	}
}

// type WebViewUser_Server interface {
func (wv *webViewUser) ListServices(ctx context.Context, call capnp_service_registry.WebViewUser_listServices) error {
	fut, release := wv.serviceView.ListServices(ctx, func(sv capnp_service_registry.ServiceViewer_listServices_Params) error {
		return nil
	})
	defer release()
	result, err := fut.Struct()
	if err != nil {
		return err
	}
	if result.HasServices() {

		// get services from serviceView
		list, err := result.Services()
		if err != nil {
			return err
		}
		callResults, err := call.AllocResults()
		if err != nil {
			return err
		}
		listlen := list.Len()
		serviceList, err := callResults.NewServices(int32(listlen))

		if err != nil {
			return err
		}
		for i := 0; i < list.Len(); i++ {
			service := list.At(i)
			if err != nil {
				return err
			}
			ref, err := capnp_service_registry.NewServiceReference(serviceList.Segment())
			if err != nil {
				return err
			}
			serviceID, err := service.ServiceID()
			if err != nil {
				serviceID = "<unknown>"
			}
			serviceType, err := service.ServiceType()
			if err != nil {
				serviceType = "<unknown>"
			}
			serviceName, err := service.ServiceName()
			if err != nil {
				serviceName = "<unknown>"
			}
			serviceDescription, err := service.ServiceDescription()
			if err != nil {
				serviceDescription = "<unknown>"
			}
			err = ref.SetServiceID(serviceID)
			if err != nil {
				return nil
			}
			ref.SetServiceType(serviceType)
			if err != nil {
				return nil
			}
			ref.SetServiceName(serviceName)
			if err != nil {
				return nil
			}
			ref.SetServiceDescription(serviceDescription)
			if err != nil {
				return nil
			}

			err = serviceList.Set(i, ref)
			if err != nil {
				return nil
			}
		}
		callResults.SetServices(serviceList)
	}

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

type storeMsg struct {
	sturdyRef string
	payload   string
	serviceId string
	answer    chan error
}

func (wv *webViewUser) StoreSturdyRef(store storeMsg) {
	// store message
	sturdyRef := store.sturdyRef
	payload := store.payload
	serviceId := store.serviceId
	// store sturdyRef in storage
	fut, release := wv.storageEditor.AddSturdyRef(context.Background(), func(params capnp_service_registry.StorageEditor_addSturdyRef_Params) error {
		storeSturdyRef, err := params.NewSturdyref()
		err = storeSturdyRef.SetPayload(payload)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetSturdyRefID(sturdyRef)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetUsersignature(wv.userID)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetServiceID(serviceId)
		params.SetSturdyref(storeSturdyRef)
		return err
	})
	defer release()
	_, err := fut.Struct()
	if err != nil {
		store.answer <- err
	} else {
		store.answer <- nil
		// add to loadedSturdyRefs
		wv.loadedStrudyRefs = append(wv.loadedStrudyRefs, sturdyRef)
	}

	return
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
