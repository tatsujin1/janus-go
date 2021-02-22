package plugins

import (
	"github.com/tatsujin1/janus-go"
)

type AudioroomResponse struct {
	Audioroom string `json:"audioroom"`
}

type AudioroomErrorResponse struct {
	AudioroomResponse
	PluginError
}

func (err *AudioroomErrorResponse) Error() string {
	return err.PluginError.Error()
}

type AudioroomRequestFactory struct {
	PluginRequestFactory
}

func MakeAudioroomRequestFactory(adminKey string) *AudioroomRequestFactory {
	return &AudioroomRequestFactory{
		PluginRequestFactory: *NewPluginRequestFactory("janus.plugin.audioroom", adminKey),
	}
}

func (f *AudioroomRequestFactory) ListRequest() *BasePluginRequest {
	request := f.make("list")
	return &request
}

func (f *AudioroomRequestFactory) CreateRequest(room *AudioroomRoom, permanent bool, allowed []string) *AudioroomCreateRequest {
	return &AudioroomCreateRequest{
		BasePluginRequest: f.make("create"),
		Room:              room,
		Permanent:         permanent,
		Allowed:           allowed,
	}
}

func (f *AudioroomRequestFactory) EditRequest(room *AudioroomRoomForEdit, permanent bool, secret string) *AudioroomEditRequest {
	return &AudioroomEditRequest{
		BasePluginRequest: f.make("edit"),
		Room:              room,
		Permanent:         permanent,
		Secret:            secret,
	}
}

func (f *AudioroomRequestFactory) DestroyRequest(roomID int, permanent bool, secret string) *AudioroomDestroyRequest {
	return &AudioroomDestroyRequest{
		BasePluginRequest: f.make("destroy"),
		RoomID:            roomID,
		Permanent:         permanent,
		Secret:            secret,
	}
}

type AudioroomListResponse struct {
	AudioroomResponse
	Rooms []*AudioroomRoomFromListResponse `json:"list"`
}

type AudioroomCreateRequest struct {
	BasePluginRequest
	Room      *AudioroomRoom
	Permanent bool
	Allowed   []string
}

func (r *AudioroomCreateRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["permanent"] = r.Permanent
	if len(r.Allowed) > 0 {
		payload["allowed"] = r.Allowed
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type AudioroomCreateResponse struct {
	AudioroomResponse
	RoomID    int  `json:"room"`
	Permanent bool `json:"permanent"`
}

type AudioroomEditRequest struct {
	BasePluginRequest
	Room      *AudioroomRoomForEdit
	Secret    string
	Permanent bool
}

func (r *AudioroomEditRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	mergeMap(payload, r.Room.AsMap())
	return payload
}

type AudioroomEditResponse struct {
	AudioroomResponse
	RoomID int `json:"room"`
}

type AudioroomDestroyRequest struct {
	BasePluginRequest
	RoomID    int
	Secret    string
	Permanent bool
}

func (r *AudioroomDestroyRequest) Payload() map[string]interface{} {
	payload := r.BasePluginRequest.Payload()
	payload["room"] = r.RoomID
	payload["permanent"] = r.Permanent
	if r.Secret != "" {
		payload["secret"] = r.Secret
	}
	return payload
}

type AudioroomDestroyResponse struct {
	AudioroomResponse
	RoomID int `json:"room"`
}

type AudioroomRoom struct {
	Room               int    `json:"room"`
	Description        string `json:"description,omitempty"`
	IsPrivate          bool   `json:"is_private"`
	Secret             string `json:"secret,omitempty"`
	Pin                string `json:"pin,omitempty"`
	RequirePvtID       bool   `json:"require_pvtid"`
	RequireE2ee        bool   `json:"require_e2ee"`
	Publishers         int    `json:"publishers"`
	Bitrate            int    `json:"bitrate"`
	FirFreq            int    `json:"fir_freq"`
	AudioCodec         string `json:"audiocodec,omitempty"`
	OpusFec            bool   `json:"opus_fec"`
	AudioLevelExt      bool   `json:"audiolevel_ext"`
	AudioLevelEvent    bool   `json:"audiolevel_event"`
	AudioActivePackets int    `json:"audio_active_packets,omitempty"`
	AudioLevelAverage  int    `json:"audio_level_average,omitempty"`
	PlayoutDelayExt    bool   `json:"playoutdelay_ext"`
	TransportWideCCExt bool   `json:"transport_wide_cc_ext"`
	Record             bool   `json:"record"`
	RecDir             string `json:"rec_dir,omitempty"`
	LockRecord         bool   `json:"lock_record"`
	NotifyJoining      bool   `json:"notify_joining"`
}

func (r *AudioroomRoom) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}

type AudioroomRoomFromListResponse struct {
	AudioroomRoom
	PinRequired     bool `json:"pin_required"`
	MaxPublishers   int  `json:"max_publishers"`
	BitrateCap      bool `json:"bitrate_cap"`
	NumParticipants int  `json:"num_participants"`
}

type AudioroomRoomForEdit struct {
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

func (r *AudioroomRoomForEdit) AsMap() map[string]interface{} {
	m, _ := janus.StructToMap(r)
	return m
}
