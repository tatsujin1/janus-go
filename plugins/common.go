package plugins

type PluginRequest interface {
	PluginName() string
	ActionName() string
	Payload() map[string]interface{}
}

type BasePluginRequest struct {
	Plugin   string
	Action   string
	AdminKey string
}

func (r *BasePluginRequest) PluginName() string {
	return r.Plugin
}

func (r *BasePluginRequest) ActionName() string {
	return r.Action
}

func (r *BasePluginRequest) Payload() map[string]interface{} {
	m := map[string]interface{}{
		"request": r.ActionName(),
	}
	if r.AdminKey != "" {
		m["admin_key"] = r.AdminKey
	}
	return m
}

type PluginRequestFactory struct {
	Plugin   string
	AdminKey string
}

func NewPluginRequestFactory(plugin, adminKey string) *PluginRequestFactory {
	return &PluginRequestFactory{
		Plugin:   plugin,
		AdminKey: adminKey,
	}
}

func (f *PluginRequestFactory) make(action string) BasePluginRequest {
	return BasePluginRequest{
		Plugin:   f.Plugin,
		Action:   action,
		AdminKey: f.AdminKey,
	}
}

type PluginError struct {
	Code   int    `json:"error_code"`
	Reason string `json:"error"`
}

func (err *PluginError) Error() string {
	return err.Reason
}

var TypeMap = map[string]map[string]func() interface{}{
	"janus.plugin.audioroom": {
		"error":   func() interface{} { return &AudioroomErrorResponse{} },
		"list":    func() interface{} { return &AudioroomListResponse{} },
		"create":  func() interface{} { return &AudioroomCreateResponse{} },
		"edit":    func() interface{} { return &AudioroomEditResponse{} },
		"destroy": func() interface{} { return &AudioroomDestroyResponse{} },
	},
	"janus.plugin.videoroom": {
		"error":   func() interface{} { return &VideoroomErrorResponse{} },
		"list":    func() interface{} { return &VideoroomListResponse{} },
		"create":  func() interface{} { return &VideoroomCreateResponse{} },
		"edit":    func() interface{} { return &VideoroomEditResponse{} },
		"destroy": func() interface{} { return &VideoroomDestroyResponse{} },
	},
	"janus.plugin.textroom": {
		"error":   func() interface{} { return &TextroomErrorResponse{} },
		"list":    func() interface{} { return &TextroomListResponse{} },
		"create":  func() interface{} { return &TextroomCreateResponse{} },
		"edit":    func() interface{} { return &TextroomEditResponse{} },
		"destroy": func() interface{} { return &TextroomDestroyResponse{} },
	},
}

func mergeMap(a, b map[string]interface{}) {
	for k, v := range b {
		a[k] = v
	}
}
