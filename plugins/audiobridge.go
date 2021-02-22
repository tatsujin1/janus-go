package plugins

import (
	"github.com/tatsujin1/janus-go"
)

type AudiobridgeResponse struct {
	Audiobridge string `json:"Audiobridge"`
}

type AudiobridgeErrorResponse struct {
	AudiobridgeResponse
	PluginError
}

func (err *AudiobridgeErrorResponse) Error() string {
	return err.PluginError.Error()
}

type AudiobridgeRequestFactory struct {
	PluginRequestFactory
}

func MakeAudiobridgeRequestFactory(adminKey string) *AudiobridgeRequestFactory {
	return &AudiobridgeRequestFactory{
		PluginRequestFactory: *NewPluginRequestFactory("janus.plugin.Audiobridge", adminKey),
	}
}

func (f *AudiobridgeRequestFactory) ListRequest() *BasePluginRequest {
	request := f.make("list")
	return &request
}

func (f *AudiobridgeRequestFactory) CreateRequest(room *AudiobridgeRoom, permanent bool, allowed []string) *AudiobridgeCreateRequest {
	return &AudiobridgeCreateRequest{
		BasePluginRequest: f.make("create"),
		Room:              room,
		Permanent:         permanent,
		Allowed:           allowed,
	}
}

func (f *AudiobridgeRequestFactory) EditRequest(room *AudiobridgeRoomForEdit, permanent bool, secret string) *AudiobridgeEditRequest {
	return &AudiobridgeEditRequest{
		BasePluginRequest: f.make("edit"),
		Room:              room,
		Permanent:         permanent,
		Secret:            secret,
	}
}

func (f *AudiobridgeRequestFactory) DestroyRequest(roomID int, permanent bool, secret string) *AudiobridgeDestroyRequest {
	return &AudiobridgeDestroyRequest{
		BasePluginRequest: f.make("destroy"),
		RoomID:            roomID,
		Permanent:         permanent,
		Secret:            secret,
	}
}

type AudiobridgeListResponse struct {
	AudiobridgeResponse
	Rooms []*AudiobridgeRoomFromListResponse `json:"list"`
}

type AudiobridgeCreateRequest struct {
	BasePluginRequest
	Room      *AudiobridgeRoom
	Permanent bool
	Allowed   []string
}

func (r *AudiobridgeCreateRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["permanent"] = r.Permanent
	if len(r.Allowed) > 0 {
		payload["allowed"] = r.Allowed
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type AudiobridgeCreateResponse struct {
	AudiobridgeResponse
	RoomID    int  `json:"room"`
	Permanent bool `json:"permanent"`
}

type AudiobridgeEditRequest struct {
	BasePluginRequest
	Room      *AudiobridgeRoomForEdit
	Secret    string
	Permanent bool
}

func (r *AudiobridgeEditRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type AudiobridgeEditResponse struct {
	AudiobridgeResponse
	RoomID int `json:"room"`
}

type AudiobridgeDestroyRequest struct {
	BasePluginRequest
	RoomID    int
	Secret    string
	Permanent bool
}

func (r *AudiobridgeDestroyRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["room"] = r.RoomID
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	return payload
}

type AudiobridgeDestroyResponse struct {
	AudiobridgeResponse
	RoomID int `json:"room"`
}

type AudiobridgeRoom struct {
	Room                int    `json:"room"`
	Description         string `json:"description,omitempty"`
	IsPrivate           bool   `json:"is_private"`
	Secret              string `json:"secret,omitempty"`
	Pin                 string `json:"pin,omitempty"`
	SamplingRate        int    `json:"sampling_rate"`
	AudioLevelExt       bool   `json:"audiolevel_ext"`
	AudioLevelEvent     bool   `json:"audiolevel_event"`
	AudioActivePackets  int    `json:"audio_active_packets,omitempty"`
	AudioLevelAverage   int    `json:"audio_level_average,omitempty"`
	DefaultPrebuffering int    `json:"default_prebuffering"`
	Record              bool   `json:"record"`
	RecordFile          string `json:"record_file,omitempty"`
}

func (r *AudiobridgeRoom) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}

type AudiobridgeRoomFromListResponse struct {
	AudiobridgeRoom
	PinRequired     bool `json:"pin_required"`
	MaxPublishers   int  `json:"max_publishers"`
	BitrateCap      bool `json:"bitrate_cap"`
	NumParticipants int  `json:"num_participants"`
}

type AudiobridgeRoomForEdit struct {
	Room         int    `json:"room"`
	Description  string `json:"new_description,omitempty"`
	IsPrivate    bool   `json:"new_is_private"`
	Secret       string `json:"new_secret,omitempty"`
	Pin          string `json:"new_pin,omitempty"`
	RequirePvtID bool   `json:"new_require_pvtid"`
	Publishers   int    `json:"new_publishers"`
	Bitrate      int    `json:"new_bitrate"`
	FirFreq      int    `json:"new_fir_freq"`
	LockRecord   bool   `json:"new_lock_record"`
}

func (r *AudiobridgeRoomForEdit) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}
