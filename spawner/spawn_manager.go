package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/google/uuid"
)

type SpawnManager struct {
	servicesByToken     map[string]*RegisteredService   // serviceId -> RegisteredService
	servicesByServiceId map[string][]*RegisteredService // token -> RegisteredService

	incommingRegistration map[string]*awaitingRegistration // token -> awaitingRegistration
	reqWaitingForService  map[string][]*requestServiceMsg  // serviceId -> requestServiceMsgC list

	// channels
	requestServiceMsgC      chan *requestServiceMsg      // request to get a service (triggered from outside)
	registerServiceMsgC     chan *registerServiceMsg     // registration of a service (triggered from outside)
	failRegisterServiceMsgC chan *failRegisterServiceMsg // failed registration of a service (triggered from inside)
	connectionLostMsgC      chan *connectionLostMsg      // connection lost to a service (triggered from inside)
	stoppedMsgC             chan *stoppedMsg             // service stopped (trigger

	// config
	serviceConfig map[string]map[string]interface{} // serviceId -> serviceConfig(to start the service)
	spawnMode     string
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

func NewSpawnManager(serviceConfig map[string]map[string]interface{}) *SpawnManager {

	spawnMode := serviceConfig["Service"]["Mode"].(string)

	// check if spawnMode is valid
	if spawnMode != "LocalWindows" && spawnMode != "LocalUnix" && spawnMode != "Slurm" {
		panic("Invalid spawn mode")
	}

	sp := &SpawnManager{
		servicesByToken:         map[string]*RegisteredService{},
		servicesByServiceId:     map[string][]*RegisteredService{},
		incommingRegistration:   make(map[string]*awaitingRegistration),
		reqWaitingForService:    map[string][]*requestServiceMsg{},
		serviceConfig:           serviceConfig,
		requestServiceMsgC:      make(chan *requestServiceMsg),
		registerServiceMsgC:     make(chan *registerServiceMsg),
		failRegisterServiceMsgC: make(chan *failRegisterServiceMsg),
		connectionLostMsgC:      make(chan *connectionLostMsg),
		stoppedMsgC:             make(chan *stoppedMsg),
		spawnMode:               spawnMode,
	}
	go sp.messageHandlerLoop()

	return sp
}

func (sm *SpawnManager) messageHandlerLoop() {

	for {
		select {
		// handle messages
		case req := <-sm.requestServiceMsgC:
			// get service from running services or await for registration
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
				// if service is not running
				// add new service, start the registration process + timeout
				service, err := sm.addNewService(req.serviceId, sm.serviceConfig[req.serviceId])
				if err != nil {
					// missconfigured service
					req.answer <- &requestAnswer{
						service: capnp.ErrorClient(err),
						err:     err,
					}
				} else {
					// add requester to waiting list
					sm.reqWaitingForService[req.serviceId] = append(sm.reqWaitingForService[req.serviceId], req)
					go sm.spawnService(service, sm.spawnMode)
				}
			}

		case registration := <-sm.registerServiceMsgC:
			// handle incomming registration requests from services
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

		case failReg := <-sm.failRegisterServiceMsgC:
			// handle failed registration, timeout or service failure on startup
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
		case stoppedMsg := <-sm.stoppedMsgC:
			// handle stopped service (by error or normal shutdown, due to idle timeout)
			if service, ok := sm.servicesByToken[stoppedMsg.token]; ok {
				service.state = stopped
				if stoppedMsg.err != nil {
					fmt.Println("Service", service.serviceId, "stopped by error: ", stoppedMsg.err)
				} else {
					fmt.Println("Service", service.serviceId, "stopped")
				}
			}
		}

	}
}

func (sm *SpawnManager) addNewService(serviceId string, serviceConfig map[string]interface{}) (*RegisteredService, error) {

	if _, ok := sm.serviceConfig[serviceId]; !ok {
		return nil, errors.New("Service configuration not found")
	}
	startTimeout := 100 // default start timeout in seconds
	if serviceConfig["StartTimeout"] != nil {
		startTimeout = serviceConfig["StartTimeout"].(int)
		if startTimeout < 0 || startTimeout > 600 {
			return nil, errors.New("Invalid StartTimeout")
		}
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

	// start timeout for registration
	go func(token string, timeout int) {
		time.Sleep(100 * time.Second)
		msg := &failRegisterServiceMsg{
			serviceToken: token,
			err:          errors.New("Timeout"),
		}
		sm.failRegisterServiceMsgC <- msg
	}(token, startTimeout)

	return service, nil
}

// RequestService returns a service by its id
func (sm *SpawnManager) RequestService(serviceId string) capnp.Client {
	// get service from running services or await for registration
	requestServiceMsg := &requestServiceMsg{
		serviceId: serviceId,
		answer:    make(chan *requestAnswer),
	}
	// send message to spawn manager
	sm.requestServiceMsgC <- requestServiceMsg
	// if service is not started yet, this may hang for a while
	answer := <-requestServiceMsg.answer
	if answer.err != nil {
		return capnp.ErrorClient(answer.err)
	}
	return answer.service
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

// spawnService starts a new service, called from inside the spawn manager,
// returns an error if the service could not be started

func (sm *SpawnManager) spawnService(regService *RegisteredService, mode string) {

	config := sm.serviceConfig[regService.serviceId] // get service config
	token := regService.token
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

	// prepare the service
	var cmd *exec.Cmd
	if mode == "LocalWindows" {
		cmd = exec.Command("cmd", "/c", path)
	} else {
		cmd = exec.Command("bin/bash", path)
	}
	// spawn a new service with the given serviceId and serviceConfig and registration token
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVICE_NAME=%s", name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVICE_ID=%s", id))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVICE_DESCRIPTION=%s", description))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVICE_TOKEN=%s", token))
	cmd.Env = append(cmd.Env, fmt.Sprintf("SERVICE_IDLE_TIMEOUT=%d", idleTimeout))
	// start the service
	err := cmd.Start()
	if err != nil {
		// handle error
		// send message to spawn manager
		msg := &failRegisterServiceMsg{
			serviceToken: token,
			err:          err,
		}
		sm.failRegisterServiceMsgC <- msg
	}

	err = cmd.Wait()
	// send message to spawn manager
	msgStop := &stoppedMsg{
		token: token,
		err:   err,
	}
	sm.stoppedMsgC <- msgStop

}

// message types

type awaitingRegistration struct {
	service   *RegisteredService
	timestamp time.Time
	err       error
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
type stoppedMsg struct {
	token string
	err   error
}
