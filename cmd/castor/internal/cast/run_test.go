package cast

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"google.golang.org/protobuf/types/known/emptypb"

	castorv1 "github.com/stupside/castor/gen/castor/v1"
	"github.com/stupside/castor/gen/castor/v1/castorv1connect"
)

type lostWatch struct {
	castorv1connect.UnimplementedCastServiceHandler
	running      bool
	malformedEnd bool
}

func (s *lostWatch) Cast(context.Context, *castorv1.CastRequest) (*castorv1.CastResponse, error) {
	s.running = true
	return &castorv1.CastResponse{CastId: "started"}, nil
}

func (s *lostWatch) Watch(_ context.Context, _ *castorv1.WatchRequest, stream *connect.ServerStream[castorv1.WatchResponse]) error {
	if s.running {
		if s.malformedEnd {
			return stream.Send(&castorv1.WatchResponse{Update: &castorv1.WatchResponse_Ended{Ended: &castorv1.Ended{}}})
		}
		return connect.NewError(connect.CodeUnavailable, errors.New("watch lost"))
	}
	return stream.Send(&castorv1.WatchResponse{Update: &castorv1.WatchResponse_Ended{Ended: &castorv1.Ended{Result: &castorv1.Ended_Stopped{Stopped: &emptypb.Empty{}}}}})
}

func TestMalformedTerminalWatchResponseStillStopsTheCast(t *testing.T) {
	service := &lostWatch{malformedEnd: true}
	_, handler := castorv1connect.NewCastServiceHandler(service)
	srv := httptest.NewTestServer(t, handler)
	client := srv.Client()
	checked := connect.WithInterceptors(validate.NewInterceptor(validate.WithValidateResponses()))
	casts := castorv1connect.NewCastServiceClient(client, srv.URL, checked)
	target := &castorv1.Target{Target: &castorv1.Target_DeviceId{DeviceId: "device"}}
	source := &castorv1.Source{Source: &castorv1.Source_Stream{Stream: &castorv1.Stream{Url: "https://cdn.example/video"}}}
	err := Run(t.Context(), casts, target, source, LinesNone)
	if err == nil {
		t.Fatal("a malformed terminal watch response succeeded")
	}
	if service.running {
		t.Fatal("a malformed terminal watch response bypassed cast cleanup")
	}
}

func (s *lostWatch) Stop(ctx context.Context, req *castorv1.StopRequest) (*castorv1.StopResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.GetCastId() != "started" {
		return nil, errors.New("wrong cast stopped")
	}
	s.running = false
	return &castorv1.StopResponse{}, nil
}

func TestWatchFailureStopsTheCastBeforeReturningItsError(t *testing.T) {
	service := &lostWatch{}
	_, handler := castorv1connect.NewCastServiceHandler(service)
	srv := httptest.NewTestServer(t, handler)
	client := srv.Client()
	casts := castorv1connect.NewCastServiceClient(client, srv.URL)
	err := Run(t.Context(), casts, nil, nil, LinesNone)
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("Run = %v, want the original watch failure", err)
	}
	if service.running {
		t.Fatal("the failed watch left a remote cast running after the command exited")
	}
}

type capture struct{ got *[]string }

func (capture) Enabled(context.Context, slog.Level) bool { return true }
func (c capture) Handle(_ context.Context, r slog.Record) error {
	*c.got = append(*c.got, r.Message)
	return nil
}
func (c capture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c capture) WithGroup(string) slog.Handler      { return c }

func TestServerLinesGoToTheDefaultLoggerAsItIsWhenTheyArrive(t *testing.T) {
	handler := FromServer()
	var got []string
	prev := slog.Default()
	slog.SetDefault(slog.New(capture{got: &got}))
	t.Cleanup(func() { slog.SetDefault(prev) })

	_ = handler.Handle(t.Context(), slog.NewRecord(time.Now(), slog.LevelWarn, "from the server", 0))
	if len(got) != 1 || got[0] != "from the server" {
		t.Fatalf("default logger saw %v, want the server's line", got)
	}
}
