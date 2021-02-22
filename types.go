// Msg Types
//
// All messages received from the gateway are first decoded to the BaseMsg
// type. The BaseMsg type extracts the following JSON from the message:
//		{
//			"janus": <Type>,
//			"transaction": <ID>,
//			"session_id": <Session>,
//			"sender": <Handle>
//		}
// The Type field is inspected to determine which concrete type
// to decode the message to, while the other fields (ID/Session/Handle) are
// inspected to determine where the message should be delivered. Messages
// with an ID field defined are considered responses to previous requests, and
// will be passed directly to requester. Messages without an ID field are
// considered unsolicited events from the gateway and are expected to have
// both Session and Handle fields defined. They will be passed to the Events
// channel of the related Handle and can be read from there.

package janus

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var msgtypes = map[string]func() interface{}{
	"error":       func() interface{} { return &ErrorMsg{} },
	"success":     func() interface{} { return &SuccessMsg{} },
	"detached":    func() interface{} { return &DetachedMsg{} },
	"server_info": func() interface{} { return &InfoMsg{} },
	"ack":         func() interface{} { return &AckMsg{} },
	"event":       func() interface{} { return &EventMsg{} },
	"webrtcup":    func() interface{} { return &WebRTCUpMsg{} },
	"media":       func() interface{} { return &MediaMsg{} },
	"hangup":      func() interface{} { return &HangupMsg{} },
	"slowlink":    func() interface{} { return &SlowLinkMsg{} },
	"timeout":     func() interface{} { return &TimeoutMsg{} },
}

type BaseMsg struct {
	Type       string     `json:"janus"`
	ID         string     `json:"transaction"`
	Session    uint64     `json:"session_id"`
	Handle     uint64     `json:"sender"`
	PluginData PluginData `json:"plugindata"`
}

type ErrorMsg struct {
	Err ErrorData `json:"error"`
}

type ErrorData struct {
	Code   int
	Reason string
}

func (err *ErrorMsg) Error() string {
	return err.Err.Reason
}

type SuccessMsg struct {
	Data       SuccessData
	PluginData PluginData
	Session    uint64 `json:"session_id"`
	Handle     uint64 `json:"sender"`
}

type SuccessData struct {
	ID uint64
}

type DetachedMsg struct{}

type InfoMsg struct {
	Name          string
	Version       int
	VersionString string `json:"version_string"`
	Author        string
	ServerName    string `json:"server-name"`
	DataChannels  bool   `json:"data_channels"`
	IPv6          bool   `json:"ipv6"`
	LocalIP       string `json:"local-ip"`
	IceTCP        bool   `json:"ice-tcp"`
	Transports    map[string]PluginInfo
	Plugins       map[string]PluginInfo
}

type PluginInfo struct {
	Name          string
	Author        string
	Description   string
	Version       int
	VersionString string `json:"version_string"`
}

type AckMsg struct{}

type EventMsg struct {
	Plugindata PluginData
	Jsep       map[string]interface{}
	Session    uint64 `json:"session_id"`
	Handle     uint64 `json:"sender"`
}

type PluginData struct {
	Plugin string
	Data   map[string]interface{}
}

type WebRTCUpMsg struct {
	Session uint64 `json:"session_id"`
	Handle  uint64 `json:"sender"`
}

type TimeoutMsg struct {
	Session uint64 `json:"session_id"`
}

type SlowLinkMsg struct {
	Uplink bool
	Lost   int64
}

type MediaMsg struct {
	Type      string
	Receiving bool
}

type HangupMsg struct {
	Reason  string
	Session uint64 `json:"session_id"`
	Handle  uint64 `json:"sender"`
}

// Event handler types

type BaseEvent struct {
	Emitter   string
	Type      int
	Subtype   int
	Timestamp int64
	Session   uint64 `json:"session_id"`
	Handle    uint64 `json:"handle_id"`
	OpaqueID  string `json:"opaque_id"`
	Event     map[string]interface{}
}

type SessionEventBody struct {
	Name      string
	Transport map[string]interface{}
}

type SessionEvent struct {
	BaseEvent
	Event SessionEventBody `json:"event"`
}

type HandleEventBody struct {
	Name   string
	Plugin string
}

type HandleEvent struct {
	BaseEvent
	Event HandleEventBody `json:"event"`
}

type ExternalEventBody struct {
	Schema string
	Data   map[string]interface{}
}

type ExternalEvent struct {
	BaseEvent
	Event ExternalEventBody `json:"event"`
}

type JSEPInfo struct {
	Type string
	SDP  string
}

type JSEPEventBody struct {
	Owner string
	JSEP  JSEPInfo
}

type JSEPEvent struct {
	BaseEvent
	Event JSEPEventBody `json:"event"`
}

type WebRTCEvent struct {
	BaseEvent
}

type MediaEvent struct {
	BaseEvent
}

type PluginEventBody struct {
	Plugin string
	Data   map[string]interface{}
}

type PluginEvent struct {
	BaseEvent
	Event PluginEventBody `json:"event"`
}

type TransportEventBody struct {
	Transport string
	ID        string
	Data      map[string]interface{}
}

type TransportEvent struct {
	BaseEvent
	Event TransportEventBody `json:"event"`
}

type CoreEvent struct {
	BaseEvent
}

var eventTypes = map[int]func() interface{}{
	1:   func() interface{} { return &SessionEvent{} },
	2:   func() interface{} { return &HandleEvent{} },
	4:   func() interface{} { return &ExternalEvent{} },
	8:   func() interface{} { return &JSEPEvent{} },
	16:  func() interface{} { return &WebRTCEvent{} },
	32:  func() interface{} { return &MediaEvent{} },
	64:  func() interface{} { return &PluginEvent{} },
	128: func() interface{} { return &TransportEvent{} },
	256: func() interface{} { return &CoreEvent{} },
}

// plugin specific types

type TextroomPostMsg struct {
	Textroom string
	Room     int // this has changed to string in janus API
	From     string
	Date     DateTime
	Text     string
	Whisper  bool
}

// Janus date time

type DateTime struct {
	time.Time
}

const dtFormat = "2006-01-02T15:04:05Z0700" // almost like RFC3339 but zone has no : (man strftime)

func (dt *DateTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		dt.Time = time.Time{}
		return
	}
	dt.Time, err = time.Parse(dtFormat, s)
	return
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	if dt.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", dt.Time.Format(dtFormat))), nil
}

var nilTime = (time.Time{}).UnixNano()

func (dt *DateTime) IsSet() bool {
	return dt.UnixNano() != nilTime
}

func ParseMessage(data []byte) (*BaseMsg, interface{}, error) {
	var base BaseMsg
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	typeFunc, ok := msgtypes[base.Type]
	if !ok {
		return &base, nil, fmt.Errorf("unknown message type received: %s", base.Type)
	}

	msg := typeFunc()
	if err := json.Unmarshal(data, &msg); err != nil {
		return &base, nil, fmt.Errorf("json.Unmarshal %s : %w", base.Type, err)
	}

	return &base, msg, nil
}

func ParseEvent(data []byte) (interface{}, error) {
	var base BaseEvent
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	switch base.Type {
	case 16:
		return &WebRTCEvent{base}, nil
	case 32:
		return &MediaEvent{base}, nil
	case 256:
		return &CoreEvent{base}, nil
	default:
		typeFunc, ok := eventTypes[base.Type]
		if !ok {
			return nil, fmt.Errorf("unknown event type received: %d", base.Type)
		}

		msg := typeFunc()
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("json.Unmarshal %d : %w", base.Type, err)
		}

		return msg, nil
	}
}

func ParseTextroomMessage(data []byte) (*TextroomPostMsg, error) {
	var msg *TextroomPostMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return msg, nil
}
