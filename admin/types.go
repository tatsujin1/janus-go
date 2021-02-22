package admin

import (
	"encoding/json"
	"fmt"

	"github.com/tatsujin1/janus-go"
	"github.com/tatsujin1/janus-go/plugins"
)

type APIRequest interface {
	ActionName() string
	Endpoint() string
	Payload() map[string]interface{}
}

type BaseRequest struct {
	Action      string
	Transaction string
	Secret      string
}

func (r *BaseRequest) ActionName() string {
	return r.Action
}

func (r *BaseRequest) Endpoint() string {
	return ""
}

func (r *BaseRequest) Payload() map[string]interface{} {
	m := make(map[string]interface{})
	m["janus"] = r.Action
	m["transaction"] = r.Transaction
	m["admin_secret"] = r.Secret
	return m
}

type TokenRequest struct {
	BaseRequest
	Token   string
	Plugins []string
}

func (r *TokenRequest) Payload() map[string]interface{} {
	m := r.BaseRequest.Payload()
	m["token"] = r.Token
	m["plugins"] = r.Plugins
	return m
}

type MessagePluginRequest struct {
	BaseRequest
	Request plugins.PluginRequest
}

func (r *MessagePluginRequest) Payload() map[string]interface{} {
	m := r.BaseRequest.Payload()
	m["plugin"] = r.Request.PluginName()
	m["request"] = r.Request.Payload()
	return m
}

type SessionRequest struct {
	BaseRequest
	SessionID uint64
}

func (r *SessionRequest) Payload() map[string]interface{} {
	m := r.BaseRequest.Payload()
	m["session_id"] = r.SessionID
	return m
}

func (r *SessionRequest) Endpoint() string {
	return fmt.Sprintf("/%d", r.SessionID)
}

type HandleRequest struct {
	SessionRequest
	HandleID uint64
}

func (r *HandleRequest) Endpoint() string {
	return fmt.Sprintf("%s/%d", r.SessionRequest.Endpoint(), r.HandleID)
}

func (r *HandleRequest) Payload() map[string]interface{} {
	m := r.SessionRequest.Payload()
	m["handle_id"] = r.HandleID
	return m
}

type BaseAMResponse struct {
	Type string `json:"janus"`
	ID   string `json:"transaction"`
}

type ErrorAMResponse struct {
	BaseAMResponse
	Err janus.ErrorData `json:"error"`
}

func (err *ErrorAMResponse) Error() string {
	return err.Err.Reason
}

type SuccessAMResponse struct {
	BaseAMResponse
	Data map[string]interface{} `json:"data"`
}

type StoredToken struct {
	Token   string   `json:"token"`
	Plugins []string `json:"allowed_plugins"`
}

type ListTokensResponse struct {
	BaseAMResponse
	Data map[string][]*StoredToken `json:"data"`
}

type ListSessionsResponse struct {
	BaseAMResponse
	Sessions []uint64 `json:"sessions"`
}

type MessagePluginResponse struct {
	BaseAMResponse
	Response map[string]interface{} `json:"response"`
}

type SessionResponse struct {
	BaseAMResponse
	SessionID uint64 `json:"session_id"`
}

type ListHandlesResponse struct {
	SessionResponse
	Handles []uint64 `json:"handles"`
}

type HandleResponse struct {
	SessionResponse
	HandleID uint64 `json:"handle_id"`
}

type HandleInfoResponse struct {
	HandleResponse
	Info map[string]interface{} `json:"info"`
}

var amResponseTypes = map[string]func() interface{}{
	"success":        func() interface{} { return &SuccessAMResponse{} },
	"error":          func() interface{} { return &ErrorAMResponse{} },
	"list_tokens":    func() interface{} { return &ListTokensResponse{} },
	"list_sessions":  func() interface{} { return &ListSessionsResponse{} },
	"message_plugin": func() interface{} { return &MessagePluginResponse{} },
	"list_handles":   func() interface{} { return &ListHandlesResponse{} },
	"handle_info":    func() interface{} { return &HandleInfoResponse{} },
}

func ParseAMResponse(r APIRequest, data []byte) (interface{}, error) {
	var base BaseAMResponse
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	typeStr := base.Type
	if typeStr == "success" {
		typeStr = r.ActionName()
	}

	typeFunc, ok := amResponseTypes[typeStr]
	if !ok {
		if base.Type == "success" {
			typeFunc = amResponseTypes["success"]
		} else {
			return nil, fmt.Errorf("unknown admin / monitor API type received: %s", typeStr)
		}
	}

	resp := typeFunc()
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("json.Unmarshal %s : %w", typeStr, err)
	}

	if mpRequest, ok := r.(*MessagePluginRequest); ok {
		if pluginTypes, ok := plugins.TypeMap[mpRequest.Request.PluginName()]; ok {
			actionName := mpRequest.Request.ActionName()
			innerPayload := resp.(*MessagePluginResponse).Response
			if _, ok := innerPayload["error"]; ok {
				actionName = "error"
			}
			if typeFunc, ok = pluginTypes[actionName]; ok {
				b, err := json.Marshal(innerPayload)
				if err != nil {
					return nil, fmt.Errorf("json.Marshal message_plugin response : %w", err)
				}

				resp = typeFunc()
				if err := json.Unmarshal(b, &resp); err != nil {
					return nil, fmt.Errorf("json.Unmarshal %s : %w", typeStr, err)
				}
			}
		}
	}

	return resp, nil
}
