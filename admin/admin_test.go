package admin

import (
	"fmt"
	"testing"

	"github.com/tatsujin1/janus-go"
	"github.com/tatsujin1/janus-go/plugins"
)

func TestAdminTokens(t *testing.T) {
	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	resp, err := api.AddToken("test-token", []string{"janus.plugin.videoroom"})
	noError(t, err)
	if resp == nil {
		t.Errorf("resp is nil")
	}

	st := findToken(t, api, "test-token")
	if st == nil {
		t.Errorf("token not in list response")
	} else {
		if len(st.Plugins) != 1 {
			t.Errorf("expecting 1 plugin got %d", len(st.Plugins))
			return
		}
		if st.Plugins[0] != "janus.plugin.videoroom" {
			t.Errorf("expecting plugin %s != %s", "janus.plugin.videoroom", st.Plugins[0])
		}
	}

	resp, err = api.AllowToken("test-token", []string{"janus.plugin.echotest"})
	noError(t, err)

	st = findToken(t, api, "test-token")
	if st == nil {
		t.Errorf("token not in list response")
	} else {
		if len(st.Plugins) != 2 {
			t.Errorf("expecting 2 plugin got %d", len(st.Plugins))
		}
	}

	resp, err = api.DisallowToken("test-token", []string{"janus.plugin.videoroom"})
	noError(t, err)

	st = findToken(t, api, "test-token")
	if st == nil {
		t.Errorf("token not in list response")
	} else {
		if len(st.Plugins) != 1 {
			t.Errorf("expecting 1 plugin got %d", len(st.Plugins))
			return
		}
		if st.Plugins[0] != "janus.plugin.echotest" {
			t.Errorf("expecting plugin %s != %s", "janus.plugin.echotest", st.Plugins[0])
		}
	}

	resp, err = api.RemoveToken("test-token")
	noError(t, err)

	st = findToken(t, api, "test-token")
	if st != nil {
		t.Errorf("token in list response after removal")
	}
}

func findToken(t *testing.T, api AdminAPI, token string) *StoredToken {
	resp, err := api.ListTokens()
	noError(t, err)

	tokens, ok := resp.(*ListTokensResponse)
	if !ok {
		t.Errorf("wrong type: ListTokensResponse != %v", resp)
		return nil
	}

	for _, x := range tokens.Data["tokens"] {
		if x.Token == token {
			return x
		}
	}

	return nil
}

func TestDefaultAdminAPI_ListSessions(t *testing.T) {
	client, err := janus.Connect("ws://localhost:8188/")
	noError(t, err)
	defer client.Close()

	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	_, err = api.AddToken("test-token", []string{})
	noError(t, err)
	defer api.RemoveToken("test-token")
	client.Token = "test-token"

	session, err := client.Create()
	noError(t, err)
	defer session.Destroy()

	resp, err := api.ListSessions()
	noError(t, err)

	tResp, ok := resp.(*ListSessionsResponse)
	if !ok {
		t.Errorf("wrong type: ListSessionsResponse != %v", resp)
		return
	}
	if len(tResp.Sessions) != 1 {
		t.Errorf("expecting exactly 1 session, found %d", len(tResp.Sessions))
		return
	}

	if session.ID != tResp.Sessions[0] {
		t.Errorf("sessionID mismatch, expected %d got %d", session.ID, tResp.Sessions[0])
		return
	}
}

func TestDefaultAdminAPI_MessagePlugin_Videoroom(t *testing.T) {
	client, err := janus.Connect("ws://localhost:8188/")
	noError(t, err)
	defer client.Close()

	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	_, err = api.AddToken("test-token", []string{})
	noError(t, err)
	defer api.RemoveToken("test-token")
	client.Token = "test-token"

	requestFactory := plugins.MakeVideoroomRequestFactory("supersecret")

	room := &plugins.VideoroomRoom{
		Room:          88,
		Description:   "test videoroom",
		IsPrivate:     false,
		Secret:        "test_secret",
		Pin:           "123456",
		Publishers:    25,
		Bitrate:       128000,
		AudioCodec:    "opus",
		VideoCodec:    "h264",
		H264Profile:   "42e01f",
		NotifyJoining: true,
	}
	resp, err := api.MessagePlugin(requestFactory.CreateRequest(room, false, nil))
	noError(t, err)

	tResp2, ok := resp.(*plugins.VideoroomCreateResponse)
	if !ok {
		t.Errorf("wrong type: VideoroomCreateResponse != %v", resp)
		return
	}
	if tResp2.RoomID != room.Room {
		t.Error("RoomID mismatch")
	}

	r := findVideoroom(t, api, requestFactory, room.Room)
	if r == nil {
		t.Error("Videoroom not found")
	} else {
		if r.Description != room.Description {
			t.Error("Videoroom description mismatch")
		}
		if !r.PinRequired {
			t.Error("Videoroom PinRequired is expected to be true")
		}
		if r.MaxPublishers != room.Publishers {
			t.Error("Videoroom Publishers mismatch")
		}
		if r.Bitrate != room.Bitrate {
			t.Error("Videoroom Bitrate mismatch")
		}
		if r.AudioCodec != room.AudioCodec {
			t.Error("Videoroom AudioCodec mismatch")
		}
		if r.VideoCodec != room.VideoCodec {
			t.Error("Videoroom VideoCodec mismatch")
		}
	}

	editRoom := &plugins.VideoroomRoomForEdit{
		Room:         room.Room,
		Description:  fmt.Sprintf("%s edit", room.Description),
		Secret:       fmt.Sprintf("%s edit", room.Secret),
		Pin:          fmt.Sprintf("%s edit", room.Pin),
		RequirePvtID: !room.RequirePvtID,
		Publishers:   room.Publishers + 1,
		Bitrate:      room.Bitrate + 1000,
		FirFreq:      room.FirFreq + 10,
		LockRecord:   !room.LockRecord,
	}

	editRoom.Room++
	resp, err = api.MessagePlugin(requestFactory.EditRequest(editRoom, false, room.Secret))
	if err == nil {
		t.Error("expecting err on edit of non existing videoroom")
	}
	editRoom.Room--
	resp, err = api.MessagePlugin(requestFactory.EditRequest(editRoom, false, room.Secret))
	noError(t, err)
	r = findVideoroom(t, api, requestFactory, room.Room)
	if r == nil {
		t.Error("Edited Videoroom not found")
	} else {
		if r.Description != editRoom.Description {
			t.Error("Videoroom description mismatch")
		}
		if r.MaxPublishers != editRoom.Publishers {
			t.Error("Videoroom Publishers mismatch")
		}
		if r.Bitrate != editRoom.Bitrate {
			t.Error("Videoroom Bitrate mismatch")
		}
		if r.RequirePvtID != editRoom.RequirePvtID {
			t.Error("Videoroom RequirePvtID mismatch")
		}
		if r.FirFreq != editRoom.FirFreq {
			t.Error("Videoroom FirFreq mismatch")
		}

		// janus doesn't support flipping of these values
		//if r.LockRecord != editRoom.LockRecord {
		//	t.Error("Videoroom LockRecord mismatch")
		//}
	}

	resp, err = api.MessagePlugin(requestFactory.DestroyRequest(room.Room+1, false, editRoom.Secret))
	if err == nil {
		t.Error("expecting err on destroy of non existing videoroom")
	}
	resp, err = api.MessagePlugin(requestFactory.DestroyRequest(room.Room, false, editRoom.Secret))
	noError(t, err)
	r = findVideoroom(t, api, requestFactory, room.Room)
	if r != nil {
		t.Error("Destroyed Videoroom is not expected to be listed")
	}
}

func findVideoroom(t *testing.T, api AdminAPI, requestFactory *plugins.VideoroomRequestFactory, room int) *plugins.VideoroomRoomFromListResponse {
	resp, err := api.MessagePlugin(requestFactory.ListRequest())
	noError(t, err)

	tResp, ok := resp.(*plugins.VideoroomListResponse)
	if !ok {
		t.Errorf("wrong type: VideoroomListResponse != %v", resp)
		return nil
	}

	for _, x := range tResp.Rooms {
		if x.Room == room {
			return x
		}
	}

	return nil
}

func TestDefaultAdminAPI_MessagePlugin_Textroom(t *testing.T) {
	client, err := janus.Connect("ws://localhost:8188/")
	noError(t, err)
	defer client.Close()

	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	_, err = api.AddToken("test-token", []string{})
	noError(t, err)
	defer api.RemoveToken("test-token")
	client.Token = "test-token"

	requestFactory := plugins.MakeTextroomRequestFactory("supersecret")

	room := &plugins.TextroomRoom{
		Room:        88,
		Description: "test textroom",
		IsPrivate:   false,
		Secret:      "test_secret",
		Pin:         "123456",
		Post:        "https://textroom.example.com/post",
	}
	resp, err := api.MessagePlugin(requestFactory.CreateRequest(room, false, nil))
	noError(t, err)

	tResp2, ok := resp.(*plugins.TextroomCreateResponse)
	if !ok {
		t.Errorf("wrong type: TextroomCreateResponse != %v", resp)
		return
	}
	if tResp2.RoomID != room.Room {
		t.Error("RoomID mismatch")
	}

	r := findTextroom(t, api, requestFactory, room.Room)
	if r == nil {
		t.Error("Textroom not found")
	} else {
		if r.Description != room.Description {
			t.Error("Textroom description mismatch")
		}
		if !r.PinRequired {
			t.Error("Textroom PinRequired is expected to be true")
		}
	}

	editRoom := &plugins.TextroomRoomForEdit{
		Room:        room.Room,
		Description: fmt.Sprintf("%s edit", room.Description),
		Secret:      fmt.Sprintf("%s edit", room.Secret),
		Pin:         fmt.Sprintf("%s edit", room.Pin),
		Post:        fmt.Sprintf("%s/edit", room.Post),
	}

	editRoom.Room++
	resp, err = api.MessagePlugin(requestFactory.EditRequest(editRoom, false, room.Secret))
	if err == nil {
		t.Error("expecting err on edit of non existing textroom")
	}
	editRoom.Room--
	resp, err = api.MessagePlugin(requestFactory.EditRequest(editRoom, false, room.Secret))
	noError(t, err)
	r = findTextroom(t, api, requestFactory, room.Room)
	if r == nil {
		t.Error("Edited Textroom not found")
	} else {
		if r.Description != editRoom.Description {
			t.Error("Textroom description mismatch")
		}
	}

	resp, err = api.MessagePlugin(requestFactory.DestroyRequest(room.Room+1, false, editRoom.Secret))
	if err == nil {
		t.Error("expecting err on destroy of non existing textroom")
	}
	resp, err = api.MessagePlugin(requestFactory.DestroyRequest(room.Room, false, editRoom.Secret))
	noError(t, err)
	r = findTextroom(t, api, requestFactory, room.Room)
	if r != nil {
		t.Error("Destroyed Videoroom is not expected to be listed")
	}
}

func findTextroom(t *testing.T, api AdminAPI, requestFactory *plugins.TextroomRequestFactory, room int) *plugins.TextroomRoomFromListResponse {
	resp, err := api.MessagePlugin(requestFactory.ListRequest())
	noError(t, err)

	tResp, ok := resp.(*plugins.TextroomListResponse)
	if !ok {
		t.Errorf("wrong type: TextroomListResponse != %v", resp)
		return nil
	}

	for _, x := range tResp.Rooms {
		if x.Room == room {
			return x
		}
	}

	return nil
}

func TestDefaultAdminAPI_ListHandles(t *testing.T) {
	client, err := janus.Connect("ws://localhost:8188/")
	noError(t, err)
	defer client.Close()

	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	_, err = api.AddToken("test-token", []string{})
	noError(t, err)
	defer api.RemoveToken("test-token")
	client.Token = "test-token"

	session, err := client.Create()
	noError(t, err)
	defer session.Destroy()

	handle, err := session.Attach("janus.plugin.videoroom")
	noError(t, err)
	defer handle.Detach()

	resp, err := api.ListHandles(session.ID)
	noError(t, err)
	if resp == nil {
		t.Error("resp is nil")
		return
	}
	tResp, ok := resp.(*ListHandlesResponse)
	if !ok {
		t.Errorf("wrong type: ListHandlesResponse != %v", resp)
		return
	}
	if len(tResp.Handles) != 1 {
		t.Errorf("expecting exactly 1 handle, found %d", len(tResp.Handles))
		return
	}

	if handle.ID != tResp.Handles[0] {
		t.Errorf("handleID mismatch, expected %d got %d", handle.ID, tResp.Handles[0])
		return
	}
}

func TestDefaultAdminAPI_HandleInfo(t *testing.T) {
	client, err := janus.Connect("ws://localhost:8188/")
	noError(t, err)
	defer client.Close()

	api, err := NewAdminAPI("http://localhost:7088/admin", "janus-go")
	noError(t, err)

	_, err = api.AddToken("test-token", []string{})
	noError(t, err)
	defer api.RemoveToken("test-token")
	client.Token = "test-token"

	session, err := client.Create()
	noError(t, err)
	defer session.Destroy()

	handle, err := session.Attach("janus.plugin.videoroom")
	noError(t, err)
	defer handle.Detach()

	resp, err := api.HandleInfo(session.ID, handle.ID)
	noError(t, err)
	if resp == nil {
		t.Error("resp is nil")
		return
	}
	tResp, ok := resp.(*HandleInfoResponse)
	if !ok {
		t.Errorf("wrong type: HandleInfoResponse != %v", resp)
		return
	}
	if tResp.Info == nil {
		t.Error("expected info to not be nil")
		return
	}
}

func noError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
