package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/google/uuid"
)

type SpawnManager struct {
	servicesByToken     map[string]*RegisteredService   // serviceId -> RegisteredService
	servicesByServiceId map[string][]*RegisteredService // token -> RegisteredService

	incommingRegistration map[string]*awaitingRegistration // token -> awaitingRegistration

	reqWaitingForService map[string][]*requestServiceMsg // serviceId -> requestServiceMsgC list

	serviceConfig           map[string]map[string]interface{} // serviceId -> serviceConfig(to start the service)
	requestSpawnMsgC        chan *requestSpawnMsg             // request to spawn a service (triggered from inside)
	requestServiceMsgC      chan *requestServiceMsg           // request to get a service (triggered from outside)
	registerServiceMsgC     chan *registerServiceMsg          // registration of a service (triggered from outside)
	failRegisterServiceMsgC chan *failRegisterServiceMsg      // failed registration of a service (triggered from inside)
	connectionLostMsgC      chan *connectionLostMsg           // connection lost to a service (triggered from inside)
}

type RegisteredService struct {
	serviceId       string
	token           string
	state           state
	bootstrapClient capnp.Client
}

type state int

const (
	running state = iota
	awaitRegistration
	missconfigured
	disconnected
	stopped
)

func (sm *SpawnManager) addNewService(serviceId string, serviceConfig map[string]interface{}) (*RegisteredService, error) {

	if _, ok := sm.serviceConfig[serviceId]; !ok {
		return nil, errors.New("Service configuration not found")
	}
	tokenGuid := uuid.New()
	token := tokenGuid.String()
	service := &RegisteredService{
		serviceId: serviceId,
		token:     token,
		state:     awaitRegistration,
	}
	sm.incommingRegistration[token] = &awaitingRegistration{
		service:   service,
		timestamp: time.Now(),
		err:       nil,
	}
	sm.servicesByToken[token] = service
	sm.servicesByServiceId[serviceId] = append(sm.servicesByServiceId[serviceId], service)

	return service, nil
}

func NewSpawnManager(serviceConfig map[string]map[string]interface{}) *SpawnManager {
	sp := &SpawnManager{
		servicesByToken:         map[string]*RegisteredService{},
		servicesByServiceId:     map[string][]*RegisteredService{},
		incommingRegistration:   make(map[string]*awaitingRegistration),
		reqWaitingForService:    map[string][]*requestServiceMsg{},
		serviceConfig:           serviceConfig,
		requestSpawnMsgC:        make(chan *requestSpawnMsg),
		requestServiceMsgC:      make(chan *requestServiceMsg),
		registerServiceMsgC:     make(chan *registerServiceMsg),
		failRegisterServiceMsgC: make(chan *failRegisterServiceMsg),
	}
	go sp.messageHandlerLoop()

	return sp
}

func (sm *SpawnManager) messageHandlerLoop() {

	for {
		select {
		case req := <-sm.requestSpawnMsgC:
			// spawn a new service
			service, err := sm.addNewService(req.serviceId, sm.serviceConfig[req.serviceId])
			if err != nil {
				req.answerToken <- ""
			} else {
				req.answerToken <- service.token
			}

		case req := <-sm.requestServiceMsgC:
			found := false
			if services, ok := sm.servicesByServiceId[req.serviceId]; ok && len(services) > 0 {
				// if service is running, return first running service
				for _, service := range services {
					switch service.state {
					// TODO: if multiple services are running, return the one with the least load
					case running:
						found = true
						req.answer <- &requestAnswer{
							service: service.bootstrapClient,
							err:     nil,
						}
						break
					case awaitRegistration:
						found = true
						// add to waiting list
						sm.reqWaitingForService[req.serviceId] = append(sm.reqWaitingForService[req.serviceId], req)
						break
					case missconfigured:
						found = true
						// service will never be available, return error
						req.answer <- &requestAnswer{
							service: capnp.ErrorClient(errors.New("Service missconfigured")),
							err:     errors.New("Service missconfigured"),
						}
						break
					}
				}
			}
			if !found {
				// if service is not running, add to waiting list
				sm.reqWaitingForService[req.serviceId] = append(sm.reqWaitingForService[req.serviceId], req)
				go sm.spawnService(req.serviceId, sm.serviceConfig[req.serviceId])
			}

		case registration := <-sm.registerServiceMsgC:
			if await, ok := sm.incommingRegistration[registration.serviceToken]; ok {
				if await.service.state == awaitRegistration {
					await.service.state = running
					await.service.bootstrapClient = registration.bootstrapper

					// notify awaiting requests
					for _, req := range sm.reqWaitingForService[await.service.serviceId] {
						req.answer <- &requestAnswer{
							service: registration.bootstrapper,
							err:     nil,
						}
					}
					delete(sm.reqWaitingForService, await.service.serviceId)    // remove requests from waiting list
					delete(sm.incommingRegistration, registration.serviceToken) // remove from incomming registration list
				}
			}

		case failReg := <-sm.failRegisterServiceMsgC: // triggered by timeout or service failure
			if await, ok := sm.incommingRegistration[failReg.serviceToken]; ok {
				if await.service.state == awaitRegistration {
					await.service.state = missconfigured
					await.err = failReg.err
					// notify awaiting requests, that the service is not available
					for _, req := range sm.reqWaitingForService[await.service.serviceId] {
						req.answer <- &requestAnswer{
							service: capnp.ErrorClient(failReg.err),
							err:     failReg.err,
						}
					}

					delete(sm.reqWaitingForService, await.service.serviceId) // remove requests from waiting list
					delete(sm.incommingRegistration, failReg.serviceToken)   // remove from incomming registration list
				}
			}

		case connLost := <-sm.connectionLostMsgC:
			// can only be triggered if service is running
			// handle connection lost to a service
			if service, ok := sm.servicesByToken[connLost.token]; ok {
				service.state = disconnected
				// handle different errors:
				// service crashed (by malformed input?),
				// service is not reachable (network error?),
				// service is not responding (overloaded?)
				// service is not available (shutdown?)
				if connLost.error != nil {
					fmt.Println("Connection lost to service: ", connLost.error)

					// TBD: handle different errors
				}
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

func (sm *SpawnManager) registerService(token string, service capnp.Client) {
	// handle incomming registration requests from services
	// send message to spawn manager
	msg := &registerServiceMsg{
		serviceToken: token,
		bootstrapper: service,
	}
	sm.registerServiceMsgC <- msg

}

func (sm *SpawnManager) spawnService(serviceId, mode string) error {

	// generate a registration token
	msg := &requestSpawnMsg{
		serviceId:   serviceId,
		answerToken: make(chan string),
	}
	sm.requestSpawnMsgC <- msg
	token := <-msg.answerToken

	if token == "" {
		return errors.New("Failed to spawn service")
	}
	config := sm.serviceConfig[serviceId] // get service config

	name := config["Name"].(string)
	id := config["Id"].(string)
	description := config["Description"].(string)
	path := config["Path"].(string) // path to script folder
	idleTimeout := config["IdleTimeout"].(int)

	// generate script file name
	if mode == "LocalWindows" {
		path = filepath.Join(path, fmt.Sprintf("%s_%s.bat", id, mode))
	}
	if mode == "LocalUnix" {
		path = filepath.Join(path, fmt.Sprintf("%s_%s.sh", id, mode))
	}
	if mode == "Slurm" {
		path = filepath.Join(path, fmt.Sprintf("%s_%s.sh", id, mode))
	}

	// start the service

	// spawn a new service with the given serviceId and serviceConfig and registration token
	// put it into awaitForRegistration map
	// enter start timestamp

	return nil
}

type awaitingRegistration struct {
	service   *RegisteredService
	timestamp time.Time
	err       error
}
type requestSpawnMsg struct {
	serviceId   string
	answerToken chan string // token
}

type requestServiceMsg struct {
	serviceId string
	answer    chan *requestAnswer
}

type requestAnswer struct {
	service capnp.Client
	err     error
}

type registerServiceMsg struct {
	serviceToken string
	bootstrapper capnp.Client
}

type failRegisterServiceMsg struct {
	serviceToken string
	err          error
}

type connectionLostMsg struct {
	token string
	error error
}
