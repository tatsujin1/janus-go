// Package janus is a Golang implementation of the Janus API, used to interact
// with the Janus WebRTC Gateway.
package janus

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/xid"
)

var debug = false

func unexpected(request string) error {
	return fmt.Errorf("Unexpected response received to '%s' request", request)
}

func newRequest(method string) (map[string]interface{}, chan interface{}) {
	req := make(map[string]interface{}, 8)
	req["janus"] = method
	return req, make(chan interface{})
}

// Gateway represents a connection to an instance of the Janus Gateway.
type Gateway struct {
	// Sessions is a map of the currently active sessions to the gateway.
	Sessions map[uint64]*Session

	ApiSecret string

	// Stored token to use for authentication.
	// See https://janus.conf.meetecho.com/docs/auth.html#token
	Token string

	// Access to the Sessions map should be synchronized with the Gateway.Lock()
	// and Gateway.Unlock() methods provided by the embedded sync.Mutex.
	sync.Mutex

	conn             *websocket.Conn
	transactions     map[xid.ID]chan interface{}
	transactionsUsed map[xid.ID]bool
	errors           chan error
	shutdown         chan struct{}
	sendChan         chan []byte
	writeMu          sync.Mutex
}

// Connect initiates a webscoket connection with the Janus Gateway
func Connect(wsURL string) (*Gateway, error) {
	websocket.DefaultDialer.Subprotocols = []string{"janus-protocol"}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

	if err != nil {
		return nil, err
	}

	gateway := new(Gateway)
	gateway.conn = conn
	gateway.transactions = make(map[xid.ID]chan interface{})
	gateway.transactionsUsed = make(map[xid.ID]bool)
	gateway.Sessions = make(map[uint64]*Session)
	gateway.sendChan = make(chan []byte, 100)
	gateway.errors = make(chan error)
	gateway.shutdown = make(chan struct{})

	go gateway.ping()
	go gateway.recv()
	return gateway, nil
}

// Close closes the underlying connection to the Gateway.
func (gateway *Gateway) Close() error {
	gateway.shutdown <- struct{}{}
	return gateway.conn.Close()
}

// GetErrChan returns a channels through which the caller can check and react to connectivity errors
func (gateway *Gateway) GetErrChan() chan error {
	return gateway.errors
}

func (gateway *Gateway) send(msg map[string]interface{}, transaction chan interface{}) {
	guid := xid.New()

	msg["transaction"] = guid.String()
	gateway.Lock()
	gateway.transactions[guid] = transaction
	gateway.transactionsUsed[guid] = false
	gateway.Unlock()

	if len(gateway.ApiSecret) > 0 {
		msg["apisecret"] = gateway.ApiSecret
	} else if len(gateway.Token) > 0 {
		msg["token"] = gateway.Token
	}

	data, err := json.Marshal(msg)
	if err != nil {
		select {
		case gateway.errors <- err:
		default:
			fmt.Fprintf(os.Stderr, "json.Marshal: %s\n", err)
		}
		return
	}

	gateway.writeMu.Lock()
	err = gateway.conn.WriteMessage(websocket.TextMessage, data)
	gateway.writeMu.Unlock()

	if err != nil {
		select {
		case gateway.errors <- err:
		default:
			fmt.Fprintf(os.Stderr, "conn.Write: %s\n", err)
		}

		return
	}
}

func passMsg(ch chan interface{}, msg interface{}) {
	ch <- msg
}

func (gateway *Gateway) ping() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-gateway.shutdown:
			return
		case <-ticker.C:
			err := gateway.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(20*time.Second))
			if err != nil {
				select {
				case gateway.errors <- err:
				default:
					fmt.Fprintf(os.Stderr, "janus ping error: %v\n", err)
				}
				return
			}
		}
	}
}

func (gateway *Gateway) sendloop() {

}

func (gateway *Gateway) recv() {

	for {
		// Read message from Gateway
		_, data, err := gateway.conn.ReadMessage()
		if err != nil {
			select {
			case gateway.errors <- err:
			default:
				fmt.Fprintf(os.Stderr,"conn.Read: %s\n", err)
			}

			return
		}

		// parse message
		base, msg, err := ParseMessage(data)
		if err != nil {
			fmt.Fprintf(os.Stderr,"parse error: %s\n", err)
			continue
		}

		var transactionUsed bool
		if base.ID != "" {
			id, _ := xid.FromString(base.ID)
			gateway.Lock()
			transactionUsed = gateway.transactionsUsed[id]
			gateway.Unlock()

		}

		// Pass message on from here
		if base.PluginData.Plugin != "" || transactionUsed {
			// Is this a Handle event?
			if base.Handle == 0 {
				// Error()
			} else {
				// Lookup Session
				gateway.Lock()
				session := gateway.Sessions[base.Session]
				gateway.Unlock()
				if session == nil {
					fmt.Fprintf(os.Stderr,"Unable to deliver message. Session gone?\n")
					continue
				}

				// Lookup Handle
				session.Lock()
				handle := session.Handles[base.Handle]
				session.Unlock()
				if handle == nil {
					fmt.Fprintf(os.Stderr,"Unable to deliver message. Handle gone?\n")
					continue
				}

				// Pass msg
				go passMsg(handle.Events, msg)
			}
		} else {
			id, _ := xid.FromString(base.ID)
			// Lookup Transaction
			gateway.Lock()
			transaction := gateway.transactions[id]
			switch msg.(type) {
			case *EventMsg:
				gateway.transactionsUsed[id] = true
			}
			gateway.Unlock()
			if transaction == nil {
				// Error()
			}

			// Pass msg
			go passMsg(transaction, msg)
		}
	}
}

// Info sends an info request to the Gateway.
// On success, an InfoMsg will be returned and error will be nil.
func (gateway *Gateway) Info() (*InfoMsg, error) {
	req, ch := newRequest("info")
	gateway.send(req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *InfoMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("info")
}

// Create sends a create request to the Gateway.
// On success, a new Session will be returned and error will be nil.
func (gateway *Gateway) Create() (*Session, error) {
	req, ch := newRequest("create")
	gateway.send(req, ch)

	msg := <-ch
	var success *SuccessMsg
	switch msg := msg.(type) {
	case *SuccessMsg:
		success = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Create new session
	session := new(Session)
	session.gateway = gateway
	session.ID = success.Data.ID
	session.Handles = make(map[uint64]*Handle)
	session.Events = make(chan interface{}, 2)

	// Store this session
	gateway.Lock()
	gateway.Sessions[session.ID] = session
	gateway.Unlock()

	return session, nil
}

// Session represents a session instance on the Janus Gateway.
type Session struct {
	// ID is the session_id of this session
	ID uint64

	// Handles is a map of plugin handles within this session
	Handles map[uint64]*Handle

	Events chan interface{}

	// Access to the Handles map should be synchronized with the Session.Lock()
	// and Session.Unlock() methods provided by the embeded sync.Mutex.
	sync.Mutex

	gateway *Gateway
}

func (session *Session) send(msg map[string]interface{}, transaction chan interface{}) {
	msg["session_id"] = session.ID
	session.gateway.send(msg, transaction)
}

// Attach sends an attach request to the Gateway within this session.
// plugin should be the unique string of the plugin to attach to.
// On success, a new Handle will be returned and error will be nil.
func (session *Session) Attach(plugin string) (*Handle, error) {
	req, ch := newRequest("attach")
	req["plugin"] = plugin
	session.send(req, ch)

	var success *SuccessMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *SuccessMsg:
		success = msg
	case *ErrorMsg:
		return nil, msg
	}

	handle := new(Handle)
	handle.session = session
	handle.ID = success.Data.ID
	handle.Events = make(chan interface{}, 8)

	session.Lock()
	session.Handles[handle.ID] = handle
	session.Unlock()

	return handle, nil
}

// KeepAlive sends a keep-alive request to the Gateway.
// On success, an AckMsg will be returned and error will be nil.
func (session *Session) KeepAlive() (*AckMsg, error) {
	req, ch := newRequest("keepalive")
	session.send(req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("keepalive")
}

// Destroy sends a destroy request to the Gateway to tear down this session.
// On success, the Session will be removed from the Gateway.Sessions map, an
// AckMsg will be returned and error will be nil.
func (session *Session) Destroy() (*AckMsg, error) {
	req, ch := newRequest("destroy")
	session.send(req, ch)

	var ack *AckMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		ack = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Remove this session from the gateway
	session.gateway.Lock()
	delete(session.gateway.Sessions, session.ID)
	session.gateway.Unlock()

	return ack, nil
}

// Handle represents a handle to a plugin instance on the Gateway.
type Handle struct {
	// ID is the handle_id of this plugin handle
	ID uint64

	// Type   // pub  or sub
	Type string

	//User   // Userid
	User string

	// Events is a receive only channel that can be used to receive events
	// related to this handle from the gateway.
	Events chan interface{}

	session *Session
}

func (handle *Handle) send(msg map[string]interface{}, transaction chan interface{}) {
	msg["handle_id"] = handle.ID
	handle.session.send(msg, transaction)
}

// Request sends a sync request
func (handle *Handle) Request(body interface{}) (*SuccessMsg, error) {
	req, ch := newRequest("message")
	if body != nil {
		req["body"] = body
	}
	handle.send(req, ch)

	msg := <-ch

	switch msg := msg.(type) {
	case *SuccessMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("message")
}

// Message sends a message request to a plugin handle on the Gateway.
// body should be the plugin data to be passed to the plugin, and jsep should
// contain an optional SDP offer/answer to establish a WebRTC PeerConnection.
// On success, an EventMsg will be returned and error will be nil.
func (handle *Handle) Message(body, jsep interface{}) (*EventMsg, error) {
	req, ch := newRequest("message")
	if body != nil {
		req["body"] = body
	}
	if jsep != nil {
		req["jsep"] = jsep
	}
	handle.send(req, ch)

GetMessage: // No tears..
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		goto GetMessage // ..only dreams.
	case *EventMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("message")
}

// Trickle sends a trickle request to the Gateway as part of establishing
// a new PeerConnection with a plugin.
// candidate should be a single ICE candidate, or a completed object to
// signify that all candidates have been sent:
//		{
//			"completed": true
//		}
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) Trickle(candidate interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidate"] = candidate
	handle.send(req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("trickle")
}

// TrickleMany sends a trickle request to the Gateway as part of establishing
// a new PeerConnection with a plugin.
// candidates should be an array of ICE candidates.
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) TrickleMany(candidates interface{}) (*AckMsg, error) {
	req, ch := newRequest("trickle")
	req["candidates"] = candidates
	handle.send(req, ch)

	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		return msg, nil
	case *ErrorMsg:
		return nil, msg
	}

	return nil, unexpected("trickle")
}

// Detach sends a detach request to the Gateway to remove this handle.
// On success, an AckMsg will be returned and error will be nil.
func (handle *Handle) Detach() (*AckMsg, error) {
	req, ch := newRequest("detach")
	handle.send(req, ch)

	var ack *AckMsg
	msg := <-ch
	switch msg := msg.(type) {
	case *AckMsg:
		ack = msg
	case *ErrorMsg:
		return nil, msg
	}

	// Remove this handle from the session
	handle.session.Lock()
	delete(handle.session.Handles, handle.ID)
	handle.session.Unlock()

	return ack, nil
}
