package main

import (
	"errors"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/google/uuid"
)

type SpawnManager struct {
	runningServices       map[string]capnp.Client          // serviceId -> service Capability
	incommingRegistration map[string]*awaitingRegistration // token -> awaitingRegistration
	tokenToServiceId      map[string]string                // token -> serviceId
	reqWaitingForService  map[string][]*requestServiceMsgC // serviceId -> requestServiceMsgC list

	serviceConfig           map[string]map[string]interface{} // serviceId -> serviceConfig(to start the service)
	requestSpawnMsgC        chan *requestSpawnMsgC            // request to spawn a service (triggered from inside)
	requestServiceMsgC      chan *requestServiceMsgC          // request to get a service (triggered from outside)
	registerServiceMsgC     chan *registerServiceMsgC         // registration of a service (triggered from outside)
	failRegisterServiceMsgC chan *failRegisterServiceMsgC     // failed registration of a service (triggered from inside)
}

func NewSpawnManager(serviceConfig map[string]map[string]interface{}) *SpawnManager {
	sp := &SpawnManager{
		runningServices:       make(map[string]capnp.Client),
		incommingRegistration: make(map[string]*awaitingRegistration),
		tokenToServiceId:      map[string]string{},
		reqWaitingForService:  map[string][]*requestServiceMsgC{},
		serviceConfig:         serviceConfig,
		requestSpawnMsgC:      make(chan *requestSpawnMsgC),
		requestServiceMsgC:    make(chan *requestServiceMsgC),
		registerServiceMsgC:   make(chan *registerServiceMsgC),
	}
	go sp.messageHandlerLoop()

	return sp
}

func (sm *SpawnManager) messageHandlerLoop() {

	for {
		select {
		case req := <-sm.requestSpawnMsgC:
			tokenGuid := uuid.New()
			token := tokenGuid.String()
			sm.incommingRegistration[token] = &awaitingRegistration{
				serviceId:        req.serviceId,
				registationToken: token,
				timestamp:        time.Now(),
				err:              nil,
			}
			// token to serviceId
			sm.tokenToServiceId[token] = req.serviceId
			req.answerToken <- token

		case req := <-sm.requestServiceMsgC:
			if service, ok := sm.runningServices[req.serviceId]; ok {
				req.answer <- &requestAnswer{
					service: service,
					err:     nil,
				}
			} else {
				// check if serviceId exists in serviceConfig
				if _, ok := sm.serviceConfig[req.serviceId]; !ok {
					req.answer <- &requestAnswer{
						service: capnp.ErrorClient(errors.New("Service not found")),
						err:     errors.New("Service not found"),
					}
				} else {
					// add to waiting list
					sm.reqWaitingForService[req.serviceId] = append(sm.reqWaitingForService[req.serviceId], req)
					if _, ok := sm.incommingRegistration[req.serviceId]; !ok {
						// service not running
						// spawn service
						go sm.spawnService(req.serviceId, sm.serviceConfig[req.serviceId])
						// await for registration
					}
				}
			}
		case registration := <-sm.registerServiceMsgC:
			if await, ok := sm.incommingRegistration[registration.serviceToken]; ok {
				if await.serviceId == registration.serviceId {
					sm.runningServices[registration.serviceId] = registration.bootstrapper
					// notify awaiting requests
					for _, req := range sm.reqWaitingForService[registration.serviceId] {
						req.answer <- &requestAnswer{
							service: registration.bootstrapper,
							err:     nil,
						}
					}
					delete(sm.incommingRegistration, registration.serviceToken) // remove from incomming registration list
					delete(sm.reqWaitingForService, registration.serviceId)     // remove requests from waiting list
				}
			}

		case failReg := <-sm.failRegisterServiceMsgC: // triggered by timeout or service failure
			if await, ok := sm.incommingRegistration[failReg.serviceToken]; ok {
				if await.serviceId == failReg.serviceId {
					await.err = failReg.err
					// notify awaiting requests, that the service is not available
					for _, req := range sm.reqWaitingForService[failReg.serviceId] {
						req.answer <- &requestAnswer{
							service: capnp.ErrorClient(failReg.err),
							err:     failReg.err,
						}
					}
					delete(sm.incommingRegistration, failReg.serviceToken) // remove from incomming registration list
					delete(sm.reqWaitingForService, failReg.serviceId)     // remove requests from waiting list
					delete(sm.tokenToServiceId, failReg.serviceToken)      // remove token to serviceId mapping
				}
			}

		case connLost := <-sm.connectionLostMsgC:
			// handle connection lost to a service
			// remove the service from running services
			if serviceId, ok := sm.tokenToServiceId[connLost.token]; ok {
				sm.runningServices[serviceId].Release()
				delete(sm.runningServices, serviceId)
				delete(sm.tokenToServiceId, connLost.token)
			}
		}

	}
}

// RequestService returns a service by its id
func RequestService(serviceId string) capnp.Client {
	// get service from running services or await for registration
	// if service is not started yet
	// trigger service start
	// await the service registration
	// return the service

	// this method may hang for a while
	// requires a timeout

	return capnp.ErrorClient(errors.New("Not implemented"))
}

func (sm *SpawnManager) registerService(serviceId string, service capnp.Client) {
	// handle incomming registration requests from services
	// check for valid registration token
	// store the service in running services
	// notify awaiting requests
}

func (sm *SpawnManager) spawnService(serviceId string, serviceConfig map[string]interface{}) error {

	// generate a registration token
	// spawn a new service with the given serviceId and serviceConfig and registration token
	// put it into awaitForRegistration map
	// enter start timestamp

	return nil
}

type awaitingRegistration struct {
	serviceId        string
	registationToken string
	timestamp        time.Time
	err              error
}
type requestSpawnMsgC struct {
	serviceId   string
	answerToken chan string // token
}

type requestServiceMsgC struct {
	serviceId string
	answer    chan *requestAnswer
}

type requestAnswer struct {
	service capnp.Client
	err     error
}

type registerServiceMsgC struct {
	serviceId    string
	serviceToken string
	bootstrapper capnp.Client
}

type failRegisterServiceMsgC struct {
	serviceToken string
	serviceId    string
	err          error
}
