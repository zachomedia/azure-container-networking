// Copyright Microsoft Corp.
// All rights reserved.

package common

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Azure/azure-container-networking/log"
)

// Listener object
type Listener struct {
	protocol     string
	localAddress string
	l            net.Listener
	mux          *http.ServeMux
}

// Creates a new Listener.
func NewListener(protocol string, localAddress string) (*Listener, error) {
	listener := Listener{
		protocol: protocol,
		localAddress: localAddress,
	}

	if protocol == "unix" && localAddress != "" {
		listener.localAddress = localAddress + ".sock"
	}

	listener.mux = http.NewServeMux()

	return &listener, nil
}

// Start creates the listener socket and starts the HTTP server.
func (listener *Listener) Start(errChan chan error) error {
	var err error

	// Succeed early if no socket was requested.
	if listener.localAddress == "" {
		return nil
	}

	listener.l, err = net.Listen(listener.protocol, listener.localAddress)
	if err != nil {
		log.Printf("Listener: Failed to listen %+v", err)
		return err
	}

	log.Printf("[Listener] Started listening on %s.", listener.localAddress)

	// Launch goroutine for servicing requests.
	go func() {
		errChan <- http.Serve(listener.l, listener.mux)
	}()

	return nil
}

// Stops listening for requests from libnetwork.
func (listener *Listener) Stop() {

	// Succeed early if no socket was requested.
	if listener.localAddress == "" {
		return
	}

	// Stop servicing requests.
	listener.l.Close()

	// Delete the unix socket.
	if listener.protocol == "unix" {
		os.Remove(listener.localAddress)
	}

	log.Printf("[Listener] Stopped listening on %s", listener.localAddress)
}

// Returns the HTTP mux for the listener.
func (listener *Listener) GetMux() *http.ServeMux {
	return listener.mux
}

// Registers a protocol handler.
func (listener *Listener) AddHandler(path string, handler func(http.ResponseWriter, *http.Request)) {
	listener.mux.HandleFunc(path, handler)
}

// Decodes JSON payload.
func (listener *Listener) Decode(w http.ResponseWriter, r *http.Request, request interface{}) error {
	var err error

	if r.Body == nil {
		err = fmt.Errorf("Request body is empty")
	} else {
		err = json.NewDecoder(r.Body).Decode(request)
	}

	if err != nil {
		http.Error(w, "Failed to decode request: "+err.Error(), http.StatusBadRequest)
		log.Printf("[Listener] Failed to decode request: %v\n", err.Error())
	}
	return err
}

// Encodes JSON payload.
func (listener *Listener) Encode(w http.ResponseWriter, response interface{}) error {
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		log.Printf("[Listener] Failed to encode response: %v\n", err.Error())
	}
	return err
}

// Sends an error response.
func (listener *Listener) SendError(w http.ResponseWriter, errMessage string) {
	json.NewEncoder(w).Encode(map[string]string{"Err": errMessage})
}
