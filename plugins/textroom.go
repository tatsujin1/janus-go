package plugins

import (
	"github.com/rs/xid"

	"github.com/tatsujin1/janus-go"
)

type TextroomRequest struct {
	BasePluginRequest
	Transaction string
}

func (r *TextroomRequest) Payload() map[string]interface{} {
	m := r.BasePluginRequest.Payload()
	m["transaction"] = r.Transaction
	return m
}

type TextroomResponse struct {
	Textroom string `json:"textroom"`
}

type TextroomErrorResponse struct {
	TextroomResponse
	PluginError
}

func (err *TextroomErrorResponse) Error() string {
	return err.PluginError.Error()
}

type TextroomRequestFactory struct {
	PluginRequestFactory
}

func MakeTextroomRequestFactory(adminKey string) *TextroomRequestFactory {
	return &TextroomRequestFactory{
		PluginRequestFactory: *NewPluginRequestFactory("janus.plugin.textroom", adminKey),
	}
}

func (f *TextroomRequestFactory) make(action string) TextroomRequest {
	return TextroomRequest{
		BasePluginRequest: BasePluginRequest{
			Plugin:   f.Plugin,
			Action:   action,
			AdminKey: f.AdminKey,
		},
		Transaction: xid.New().String(),
	}
}

func (f *TextroomRequestFactory) ListRequest() *TextroomRequest {
	request := f.make("list")
	return &request
}

func (f *TextroomRequestFactory) CreateRequest(room *TextroomRoom, permanent bool, allowed []string) *TextroomCreateRequest {
	return &TextroomCreateRequest{
		TextroomRequest: f.make("create"),
		Room:            room,
		Permanent:       permanent,
		Allowed:         allowed,
	}
}

func (f *TextroomRequestFactory) EditRequest(room *TextroomRoomForEdit, permanent bool, secret string) *TextroomEditRequest {
	return &TextroomEditRequest{
		TextroomRequest: f.make("edit"),
		Room:            room,
		Permanent:       permanent,
		Secret:          secret,
	}
}

func (f *TextroomRequestFactory) DestroyRequest(roomID int, permanent bool, secret string) *TextroomDestroyRequest {
	return &TextroomDestroyRequest{
		TextroomRequest: f.make("destroy"),
		RoomID:          roomID,
		Permanent:       permanent,
		Secret:          secret,
	}
}

type TextroomListResponse struct {
	TextroomResponse
	Rooms []*TextroomRoomFromListResponse `json:"list"`
}

type TextroomCreateRequest struct {
	TextroomRequest
	Room      *TextroomRoom
	Permanent bool
	Allowed   []string
}

func (r *TextroomCreateRequest) Payload() map[string]interface{} {
	payload := r.TextroomRequest.Payload()
	payload["permanent"] = r.Permanent
	if r.Allowed != "" {
		payload["allowed"] = r.Allowed
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type TextroomCreateResponse struct {
	TextroomResponse
	RoomID    int  `json:"room"`
	Permanent bool `json:"permanent"`
}

type TextroomEditRequest struct {
	TextroomRequest
	Room      *TextroomRoomForEdit
	Secret    string
	Permanent bool
}

func (r *TextroomEditRequest) Payload() map[string]interface{} {
	payload := r.TextroomRequest.Payload()
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type TextroomEditResponse struct {
	TextroomResponse
	RoomID int `json:"room"`
}

type TextroomDestroyRequest struct {
	TextroomRequest
	RoomID    int
	Secret    string
	Permanent bool
}

func (r *TextroomDestroyRequest) Payload() map[string]interface{} {
	payload := r.TextroomRequest.Payload()
	payload["room"] = r.RoomID
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	return payload
}

type TextroomDestroyResponse struct {
	TextroomResponse
	RoomID int `json:"room"`
}

type TextroomRoom struct {
	Room        int    `json:"room"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
	Secret      string `json:"secret,omitempty"`
	Pin         string `json:"pin,omitempty"`
	Post        string `json:"post,omitempty"`
}

func (r *TextroomRoom) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}

type TextroomRoomFromListResponse struct {
	TextroomRoom
	PinRequired     bool `json:"pin_required"`
	NumParticipants int  `json:"num_participants"`
}

type TextroomRoomForEdit struct {
	Room        int    `json:"room"`
	Description string `json:"new_description,omitempty"`
	IsPrivate   bool   `json:"new_is_private"`
	Secret      string `json:"new_secret,omitempty"`
	Pin         string `json:"new_pin,omitempty"`
	Post        string `json:"new_post,omitempty"`
}

func (r *TextroomRoomForEdit) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}
