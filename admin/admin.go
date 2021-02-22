package admin

import (
	"fmt"
	"strings"

	"github.com/rs/xid"

	"github.com/tatsujin1/janus-go/plugins"
)

type AdminAPI interface {
	AddToken(token string, plugins []string) (interface{}, error)
	AllowToken(token string, plugins []string) (interface{}, error)
	DisallowToken(token string, plugins []string) (interface{}, error)
	RemoveToken(token string) (interface{}, error)
	ListTokens() (interface{}, error)

	ListSessions() (interface{}, error)
	MessagePlugin(request plugins.PluginRequest) (interface{}, error)

	ListHandles(sessionID uint64) (interface{}, error)
	HandleInfo(sessionID, handleID uint64) (interface{}, error)

	Close() error
}

type DefaultAdminAPI struct {
	transport Transport
	secret    string
}

func NewAdminAPI(url, secret string) (*DefaultAdminAPI, error) {
	api := new(DefaultAdminAPI)
	api.secret = secret

	if strings.HasPrefix(url, "http") {
		api.transport = NewHttpTransport(url)
	} else {
		return nil, fmt.Errorf("unsupported transport for %s", url)
	}

	return api, nil
}

func (api *DefaultAdminAPI) AddToken(token string, plugins []string) (interface{}, error) {
	return api.transport.Request(api.makeTokenRequest("add_token", token, plugins))
}

func (api *DefaultAdminAPI) AllowToken(token string, plugins []string) (interface{}, error) {
	return api.transport.Request(api.makeTokenRequest("allow_token", token, plugins))
}

func (api *DefaultAdminAPI) DisallowToken(token string, plugins []string) (interface{}, error) {
	return api.transport.Request(api.makeTokenRequest("disallow_token", token, plugins))
}

func (api *DefaultAdminAPI) RemoveToken(token string) (interface{}, error) {
	return api.transport.Request(api.makeTokenRequest("remove_token", token, nil))
}

func (api *DefaultAdminAPI) ListTokens() (interface{}, error) {
	return api.transport.Request(api.makeBaseRequest("list_tokens"))
}

func (api *DefaultAdminAPI) ListSessions() (interface{}, error) {
	return api.transport.Request(api.makeBaseRequest("list_sessions"))
}

func (api *DefaultAdminAPI) MessagePlugin(request plugins.PluginRequest) (interface{}, error) {
	return api.transport.Request(api.makeMessagePluginRequest(request))
}

func (api *DefaultAdminAPI) ListHandles(sessionID uint64) (interface{}, error) {
	return api.transport.Request(api.makeSessionRequest("list_handles", sessionID))
}

func (api *DefaultAdminAPI) HandleInfo(sessionID, handleID uint64) (interface{}, error) {
	return api.transport.Request(api.makeHandleRequest("handle_info", sessionID, handleID))
}

func (api *DefaultAdminAPI) Close() error {
	return api.transport.Close()
}

func (api *DefaultAdminAPI) makeBaseRequest(action string) *BaseRequest {
	return &BaseRequest{
		Action:      action,
		Transaction: xid.New().String(),
		Secret:      api.secret,
	}
}

func (api *DefaultAdminAPI) makeTokenRequest(action, token string, plugins []string) *TokenRequest {
	return &TokenRequest{
		BaseRequest: *api.makeBaseRequest(action),
		Token:       token,
		Plugins:     plugins,
	}
}

func (api *DefaultAdminAPI) makeMessagePluginRequest(request plugins.PluginRequest) *MessagePluginRequest {
	return &MessagePluginRequest{
		BaseRequest: *api.makeBaseRequest("message_plugin"),
		Request:     request,
	}
}

func (api *DefaultAdminAPI) makeSessionRequest(action string, sessionID uint64) *SessionRequest {
	return &SessionRequest{
		BaseRequest: *api.makeBaseRequest(action),
		SessionID:   sessionID,
	}
}

func (api *DefaultAdminAPI) makeHandleRequest(action string, sessionID, handleID uint64) *HandleRequest {
	return &HandleRequest{
		SessionRequest: *api.makeSessionRequest(action, sessionID),
		HandleID:       handleID,
	}
}
