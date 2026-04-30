// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// switchTeamServer is a minimal Phoenix-Channel-V2 stub built for the
// SwitchTeam round-trip tests. Auto-acks phx_join + heartbeats; for
// scope.switch_team it replies with a configurable status/response so
// each test can drive a different code path.
type switchTeamServer struct {
	srv *httptest.Server

	mu      sync.Mutex
	conn    *websocket.Conn
	writeMu sync.Mutex

	// reply payload for the next scope.switch_team push.
	replyStatus   string
	replyResponse any
}

func newSwitchTeamServer(t *testing.T) *switchTeamServer {
	t.Helper()
	s := &switchTeamServer{
		replyStatus:   "ok",
		replyResponse: defaultSwitchOK("019dbd00-0000-7000-8000-000000000001"),
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *switchTeamServer) URL() string {
	u, _ := url.Parse(s.srv.URL)
	u.Scheme = "ws"
	u.Path = "/console/websocket"
	return u.String()
}

func (s *switchTeamServer) Close() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "test cleanup")
	}
	s.srv.Close()
}

// SetReply queues the reply shape returned for the next scope.switch_team
// inbound frame. status is "ok" or "error"; response is marshalled as
// the reply's `response` field.
func (s *switchTeamServer) SetReply(status string, response any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replyStatus = status
	s.replyResponse = response
}

func (s *switchTeamServer) sendReply(joinRef, ref, topic, status string, response any) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return websocket.CloseError{Code: websocket.StatusAbnormalClosure}
	}
	payload := map[string]any{"status": status, "response": response}
	jr, r := joinRef, ref
	frame, err := newFrame(&jr, &r, topic, "phx_reply", payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}

func (s *switchTeamServer) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "")
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
	}()

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var f Frame
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		ref := ""
		if f.Ref != nil {
			ref = *f.Ref
		}
		joinRef := ""
		if f.JoinRef != nil {
			joinRef = *f.JoinRef
		}
		switch f.Event {
		case "phx_join":
			_ = s.sendReply(joinRef, ref, f.Topic, "ok", map[string]any{})
		case "heartbeat":
			_ = s.sendReply("", ref, "phoenix", "ok", map[string]any{})
		case "scope.switch_team":
			s.mu.Lock()
			status, resp := s.replyStatus, s.replyResponse
			s.mu.Unlock()
			_ = s.sendReply(joinRef, ref, f.Topic, status, resp)
		}
	}
}

// defaultSwitchOK returns a typical success envelope for `scope.switch_team`,
// matching the server's lib/truestamp_web/channels/console_channel.ex
// reply shape. Tests can override this via SetReply for variant cases.
func defaultSwitchOK(targetTeamID string) map[string]any {
	return map[string]any{
		"scope": map[string]any{
			"user_id": "user-uuid",
			"team_id": targetTeamID,
			"plan":    "pro",
		},
		"team": map[string]any{
			"id":              targetTeamID,
			"name":            "Engineering",
			"personal":        false,
			"ownership_model": "creator_retains",
		},
		"role": "team_admin",
		"streams": map[string]any{
			"catalog": []string{"items.team"},
			"items":   []string{},
		},
	}
}

// connectAndJoin does the standard setup so each scope.switch_team test
// doesn't repeat the boilerplate.
func connectAndJoin(t *testing.T, s *switchTeamServer) *Client {
	t.Helper()
	c, err := New(Options{URL: s.URL(), APIKey: "test_key", HeartbeatInterval: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return c
}

func TestSwitchTeam_Success(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	const newTeamID = "019dbd00-0000-7000-8000-000000000001"
	s.SetReply("ok", defaultSwitchOK(newTeamID))

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := c.SwitchTeam(ctx, newTeamID)
	if err != nil {
		t.Fatalf("SwitchTeam: %v", err)
	}
	if reply.Scope.TeamID != newTeamID {
		t.Errorf("Scope.TeamID = %q, want %q", reply.Scope.TeamID, newTeamID)
	}
	if reply.Team.Name != "Engineering" || reply.Team.Personal {
		t.Errorf("Team attrs wrong: %+v", reply.Team)
	}
	if reply.Role != "team_admin" {
		t.Errorf("Role = %q, want team_admin", reply.Role)
	}
	if len(reply.Streams.Catalog) != 1 || reply.Streams.Catalog[0] != "items.team" {
		t.Errorf("Streams.Catalog = %v, want [items.team]", reply.Streams.Catalog)
	}
}

func TestSwitchTeam_Forbidden(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	s.SetReply("error", map[string]any{
		"code":    "forbidden",
		"message": "not a member of team xyz",
	})

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "019d99999-9999-7999-8999-999999999999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	cerr, ok := err.(*ChannelError)
	if !ok {
		t.Fatalf("err type = %T, want *ChannelError; %v", err, err)
	}
	if cerr.Code != ChannelErrCodeForbidden {
		t.Errorf("Code = %q, want %q", cerr.Code, ChannelErrCodeForbidden)
	}
	if !strings.Contains(cerr.Message, "not a member") {
		t.Errorf("Message = %q, expected to contain 'not a member'", cerr.Message)
	}
}

func TestSwitchTeam_NotFound(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	s.SetReply("error", map[string]any{
		"code":    "not_found",
		"message": "team not found",
	})

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "019dffff-ffff-7fff-8fff-ffffffffffff")
	cerr, ok := err.(*ChannelError)
	if !ok {
		t.Fatalf("err type = %T, want *ChannelError", err)
	}
	if cerr.Code != ChannelErrCodeNotFound {
		t.Errorf("Code = %q, want %q", cerr.Code, ChannelErrCodeNotFound)
	}
}

func TestSwitchTeam_Noop(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	s.SetReply("error", map[string]any{
		"code":    "noop",
		"message": "already scoped to xyz",
	})

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "019dbd00-0000-7000-8000-000000000000")
	cerr, ok := err.(*ChannelError)
	if !ok {
		t.Fatalf("err type = %T, want *ChannelError", err)
	}
	if cerr.Code != ChannelErrCodeNoop {
		t.Errorf("Code = %q, want %q", cerr.Code, ChannelErrCodeNoop)
	}
}

func TestSwitchTeam_Invalid(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	s.SetReply("error", map[string]any{
		"code":    "invalid",
		"message": "expected {team_id: <uuid>}",
	})

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "x")
	cerr, ok := err.(*ChannelError)
	if !ok {
		t.Fatalf("err type = %T, want *ChannelError", err)
	}
	if cerr.Code != ChannelErrCodeInvalid {
		t.Errorf("Code = %q, want %q", cerr.Code, ChannelErrCodeInvalid)
	}
}

func TestSwitchTeam_RateLimited(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	s.SetReply("error", map[string]any{
		"code":    "rate_limited",
		"message": "command rate limit exceeded",
	})

	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "019dbd00-0000-7000-8000-000000000001")
	cerr, ok := err.(*ChannelError)
	if !ok {
		t.Fatalf("err type = %T, want *ChannelError", err)
	}
	if cerr.Code != ChannelErrCodeRateLimited {
		t.Errorf("Code = %q, want %q", cerr.Code, ChannelErrCodeRateLimited)
	}
}

func TestSwitchTeam_RejectsEmptyID(t *testing.T) {
	t.Parallel()

	s := newSwitchTeamServer(t)
	c := connectAndJoin(t, s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.SwitchTeam(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty team_id")
	}
	if !strings.Contains(err.Error(), "team_id is required") {
		t.Errorf("err = %q, want to mention team_id required", err.Error())
	}
}

func TestChannelError_Stringer(t *testing.T) {
	cases := []struct {
		err  *ChannelError
		want string
	}{
		{&ChannelError{Code: ChannelErrCodeForbidden, Message: "no membership"},
			"forbidden: no membership"},
		{&ChannelError{Code: ChannelErrCodeNoop},
			"noop"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
	}
}
