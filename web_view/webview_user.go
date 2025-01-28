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
	storageEditor      capnp_service_registry.StorageEditor
	serviceView        capnp_service_registry.ServiceViewer
	userID             string
	loadedStrudyRefs   map[string]SturdyRefUserView // map[token]SturdyRefUserView
	loadedServiveViews map[string]capnp.Client
	storeMsgChan       chan storeMsg
	deleteMsgChan      chan deleteMsg
	listSturdyRefChan  chan getSturdyRefList
}

type SturdyRefUserView struct {
	sturdyRefToken string
	serviceID      string
	fullSturdyRef  string
	desc           string
}

func newWebViewUser(storeEditorCap, viewerCap capnp.Client, userId string) *webViewUser {
	wv := &webViewUser{
		storageEditor:     capnp_service_registry.StorageEditor(storeEditorCap),
		serviceView:       capnp_service_registry.ServiceViewer(viewerCap),
		userID:            userId,
		loadedStrudyRefs:  make(map[string]SturdyRefUserView),
		storeMsgChan:      make(chan storeMsg),
		deleteMsgChan:     make(chan deleteMsg),
		listSturdyRefChan: make(chan getSturdyRefList),
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
		case deleteMsg := <-wv.deleteMsgChan:
			err := wv.deleteSturdyRef(deleteMsg.sturdyRef)
			deleteMsg.answer <- err
		case getSturdyRefList := <-wv.listSturdyRefChan:
			sturdyRefs, _ := wv.sturdyRefFromStorage() // TODO: handle error... maybe do not forward every internal error to the client -.-'
			getSturdyRefList.answer <- sturdyRefs
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
				return err
			}
			err = ref.SetServiceType(serviceType)
			if err != nil {
				return err
			}
			err = ref.SetServiceName(serviceName)
			if err != nil {
				return err
			}
			err = ref.SetServiceDescription(serviceDescription)
			if err != nil {
				return err
			}

			err = serviceList.Set(i, ref)
			if err != nil {
				return err
			}
		}
		return callResults.SetServices(serviceList)
	}

	return nil
}

func (wv *webViewUser) GetServiceView(ctx context.Context, call capnp_service_registry.WebViewUser_getServiceView) error {

	return nil
}

func (wv *webViewUser) RemoveSturdyRef(ctx context.Context, call capnp_service_registry.WebViewUser_removeSturdyRef) error {
	srToken, err := call.Args().SturdyRef()
	if err != nil {
		return err
	}
	if srToken == "" {
		return errors.New("no sturdy ref provided")
	}
	deleteMsg := deleteMsg{
		sturdyRef: srToken,
		answer:    make(chan error),
	}
	wv.deleteMsgChan <- deleteMsg
	err = <-deleteMsg.answer

	return err
}

func (wv *webViewUser) ListSturdyRefs(ctx context.Context, call capnp_service_registry.WebViewUser_listSturdyRefs) error {

	// list all sturdyRefs from storage
	getSturdyRefList := getSturdyRefList{
		answer: make(chan map[string]SturdyRefUserView),
	}
	sturdyRefs := <-getSturdyRefList.answer
	results, err := call.AllocResults()
	if err != nil {
		return err
	}
	list, err := results.NewSturdyRefs(int32(len(sturdyRefs)))
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(sturdyRefs))
	for k := range sturdyRefs {
		keys = append(keys, k)
	}
	for i, sturdyRefToken := range keys {
		sturdyRefStored, err := capnp_service_registry.NewSturdyRefUserView(list.Segment())
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetSturdyRefID(sturdyRefs[sturdyRefToken].fullSturdyRef)
		if err != nil {
			return err
		}
		err = sturdyRefStored.SetServiceID(sturdyRefs[sturdyRefToken].serviceID)
		if err != nil {
			return err
		}

		err = sturdyRefStored.SetPayloaDescription(sturdyRefs[sturdyRefToken].desc)
		if err != nil {
			return err
		}
		err = list.Set(i, sturdyRefStored)
		if err != nil {
			return err
		}
	}
	return results.SetSturdyRefs(list)
}

type getSturdyRefList struct {
	answer chan map[string]SturdyRefUserView
}

func (wv *webViewUser) sturdyRefFromStorage() (map[string]SturdyRefUserView, error) {

	if wv.loadedStrudyRefs != nil {
		return wv.loadedStrudyRefs, nil
	} else {
		// load sturdyRefs from storage
		fut, release := wv.storageEditor.ListSturdyRefs(context.Background(), func(p capnp_service_registry.StorageEditor_listSturdyRefs_Params) error {
			return p.SetUsersignature(wv.userID)
		})
		defer release()
		result, err := fut.Struct()
		if err != nil {
			return nil, err
		}
		if result.HasSturdyrefs() {
			sturdyRefList, err := result.Sturdyrefs()
			if err != nil {
				return nil, err
			}

			for i := 0; i < sturdyRefList.Len(); i++ {
				sturdyRef := sturdyRefList.At(i)
				if err != nil {
					return nil, err
				}
				sturdyRefID, err := sturdyRef.SturdyRefID()
				if err != nil {
					return nil, err
				}
				serviceId, err := sturdyRef.ServiceID()
				if err != nil {
					return nil, err
				}
				desc := ""
				if sturdyRef.HasPayloaDescription() {
					desc, err = sturdyRef.PayloaDescription()
					if err != nil {
						return nil, err
					}
				}

				wv.loadedStrudyRefs[sturdyRefID] = SturdyRefUserView{
					sturdyRefToken: sturdyRefID,
					serviceID:      serviceId,
					fullSturdyRef:  getFullSturdyRef(sturdyRefID),
					desc:           desc,
				}
			}
			return wv.loadedStrudyRefs, nil
		}
	}
	return wv.loadedStrudyRefs, nil
}

// TODO implement getFullSturdyRef
func getFullSturdyRef(sturdyRefToken string) string {
	// get full sturdyRef from serviceView
	return "someurl" + sturdyRefToken
}

type deleteMsg struct {
	sturdyRef string
	answer    chan error
}

func (wv *webViewUser) deleteSturdyRef(sturdyRef string) error {
	// delete sturdyRef from storage

	fut, release := wv.storageEditor.DeleteSturdyRef(context.Background(), func(p capnp_service_registry.StorageEditor_deleteSturdyRef_Params) error {
		return p.SetSturdyRefID(sturdyRef)
	})
	defer release()
	_, err := fut.Struct()
	if err != nil {
		return err
	}

	delete(wv.loadedStrudyRefs, sturdyRef)

	return nil
}

type storeMsg struct {
	sturdyRefToken string
	payload        string
	serviceId      string
	desc           string
	answer         chan error
}

func (wv *webViewUser) StoreSturdyRef(store storeMsg) {
	// store message
	sturdyRefToken := store.sturdyRefToken
	payload := store.payload
	serviceId := store.serviceId
	desc := store.desc

	// store sturdyRef in storage
	fut, release := wv.storageEditor.AddSturdyRef(context.Background(), func(params capnp_service_registry.StorageEditor_addSturdyRef_Params) error {
		storeSturdyRef, err := params.NewSturdyref()
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetPayload(payload)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetSturdyRefID(sturdyRefToken)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetUsersignature(wv.userID)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetServiceID(serviceId)
		if err != nil {
			return err
		}
		err = storeSturdyRef.SetPayloaDescription(desc)
		if err != nil {
			return err
		}
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
		wv.loadedStrudyRefs[sturdyRefToken] = SturdyRefUserView{
			sturdyRefToken: sturdyRefToken,
			serviceID:      serviceId,
			fullSturdyRef:  getFullSturdyRef(sturdyRefToken),
			desc:           desc,
		}
	}
}

type saveCallback struct {
	serviceId string
	webView   *webViewUser
}

func (wv *webViewUser) newSaveCallback(serviceId string) *saveCallback {
	return &saveCallback{
		serviceId: serviceId,
		webView:   wv,
	}
}

func (sv *saveCallback) Save(ctx context.Context, call capnp_service_registry.SaveCallback_save_Params) error {
	payload, err := call.Payload()
	if err != nil {
		return err
	}
	if payload == "" {
		return errors.New("payload is empty")
	}
	desc := ""
	if call.HasDesc() {
		desc, err = call.Desc()
		if err != nil {
			return err
		}
	}
	// store sturdyRef in storage
	store := storeMsg{
		sturdyRefToken: "",
		payload:        payload,
		serviceId:      sv.serviceId,
		desc:           desc,
		answer:         make(chan error),
	}
	sv.webView.storeMsgChan <- store
	err = <-store.answer
	return err
}

// interface for capability forwarding handler (from commonlib)
type restoreWebViewHandler struct {
	userEditor capnp.Client
	storedcap  capnp.Client
	viewerCap  capnp.Client

	loadedCaps      map[string]*webViewUser
	getCap          chan getCap
	deleteSturdyRef chan deleteStrudyRef
}

func newRestoreWebViewHandler(storecap, viewerCap, userEditor capnp.Client) *restoreWebViewHandler {
	handler := &restoreWebViewHandler{
		userEditor:      userEditor,
		storedcap:       storecap,
		viewerCap:       viewerCap,
		loadedCaps:      make(map[string]*webViewUser),
		getCap:          make(chan getCap),
		deleteSturdyRef: make(chan deleteStrudyRef),
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
			case delete := <-handler.deleteSturdyRef:
				if userWV, ok := handler.loadedCaps[delete.user]; ok {
					// delete sturdyRef from loadedCaps
					deleteSR := deleteMsg{
						sturdyRef: delete.sturdyRef,
						answer:    make(chan error),
					}
					userWV.deleteMsgChan <- deleteSR
					err := <-deleteSR.answer
					delete.answer <- deleteStrudyRefAnswer{success: true, err: err}
				} else {
					delete.answer <- deleteStrudyRefAnswer{success: false, err: nil}
				}
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

type deleteStrudyRef struct {
	user      string
	sturdyRef string
	answer    chan deleteStrudyRefAnswer
}
type deleteStrudyRefAnswer struct {
	success bool
	err     error
}
