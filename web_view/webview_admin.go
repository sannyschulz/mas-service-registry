package main

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	"github.com/sannyschulz/mas-service-registry/capnp_service_registry"
	"github.com/zalf-rpm/mas-infrastructure/src/go/commonlib"
)

// implement interface WebViewUser
type webViewAdmin struct {
	storageEditor         capnp_service_registry.StorageEditor
	serviceView           capnp_service_registry.ServiceViewer
	userEditor            capnp_service_registry.UserEditor
	persistable           *commonlib.Persistable
	userWebViewRestoreHdl *restoreWebViewHandler
}

func newWebViewAdmin(restorer *commonlib.Restorer, storecap, viewerCap, userEditorCap *capnp.Client) *webViewAdmin {
	wv := &webViewAdmin{
		persistable:           commonlib.NewPersistable(restorer),
		storageEditor:         capnp_service_registry.StorageEditor(*storecap),
		serviceView:           capnp_service_registry.ServiceViewer(*viewerCap),
		userEditor:            capnp_service_registry.UserEditor(*userEditorCap),
		userWebViewRestoreHdl: newRestoreWebViewHandler(*storecap, *viewerCap, *userEditorCap),
	}

	restoreFunc := func() capnp.Client {
		return capnp.Client(capnp_service_registry.WebViewAdmin_ServerToClient(wv))
	}
	wv.persistable.Cap = restoreFunc
	return wv
}

// WebViewAdmin_Server interface

func (wv *webViewAdmin) RemoveSturdyRef(ctx context.Context, call capnp_service_registry.WebViewAdmin_removeSturdyRef) error {
	stRef, err := call.Args().SturdyRef()
	if err != nil {
		return err
	}
	user, err := call.Args().Usersignature()
	if err != nil {
		return err
	}
	if user == "" {
		return errors.New("no user signature provided")
	}
	if stRef == "" {
		return errors.New("no sturdy ref provided")
	}
	deleteStrudyRefForUser := deleteStrudyRef{
		sturdyRef: stRef,
		user:      user,
		answer:    make(chan deleteStrudyRefAnswer),
	}
	wv.userWebViewRestoreHdl.deleteSturdyRef <- deleteStrudyRefForUser
	answer := <-deleteStrudyRefForUser.answer
	if answer.success && answer.err != nil {
		return answer.err
	} else if !answer.success {
		fut, release := wv.storageEditor.DeleteSturdyRef(ctx, func(p capnp_service_registry.StorageEditor_deleteSturdyRef_Params) error {
			return p.SetSturdyRefID(stRef)
		})
		defer release()
		_, err = fut.Struct()
		if err != nil {
			return err
		}
	}

	return nil
}

func (wv *webViewAdmin) ListAllSturdyRefs(ctx context.Context, call capnp_service_registry.WebViewAdmin_listAllSturdyRefs) error {
	return nil
}

func (wv *webViewAdmin) NewWebViewUser(ctx context.Context, call capnp_service_registry.WebViewAdmin_newWebViewUser) error {

	return nil
}
