package natsgw

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/CyCoreSystems/ari"
	"github.com/CyCoreSystems/ari/client/mock"
	"github.com/CyCoreSystems/ari/client/nc"
	"github.com/golang/mock/gomock"
)

func TestPlaybackData(t *testing.T) {

	//TODO: embed nats?

	bin, err := exec.LookPath("gnatsd")
	if err != nil {
		t.Skip("No gnatsd binary in PATH, skipping")
	}

	cmd := exec.Command(bin, "-p", "4333")
	if err := cmd.Start(); err != nil {
		t.Errorf("Unable to run gnatsd: '%v'", err)
		return
	}

	defer func() {
		cmd.Process.Signal(syscall.SIGTERM)
	}()

	<-time.After(ServerWaitDelay)

	// test client

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var playbackData ari.PlaybackData
	var playbackErrorMessage = "Could not find playback"

	mockPlayback := mock.NewMockPlayback(ctrl)
	mockPlayback.EXPECT().Data("pb1").Return(playbackData, nil)
	mockPlayback.EXPECT().Data("pb2").Return(playbackData, errors.New(playbackErrorMessage))

	cl := &ari.Client{
		Playback: mockPlayback,
	}
	s, err := NewServer(cl, &Options{
		URL: "nats://127.0.0.1:4333",
	})

	failed := s == nil || err != nil
	if failed {
		t.Errorf("natsgw.NewServer(cl, nil) => {%v, %v}, expected {%v, %v}", s, err, "cl", "nil")
	}

	go s.Listen()
	defer s.Close()

	natsClient, err := nc.New("nats://127.0.0.1:4333")

	failed = natsClient == nil || err != nil
	if failed {
		t.Errorf("nc.New(url) => {%v, %v}, expected {%v, %v}", natsClient, err, "cl", "nil")
	}

	{
		ret, err := natsClient.Playback.Data("pb1")
		failed = err != nil
		if failed {
			t.Errorf("nc.Playback.Data('pb1') => ('%v','%v'), expected ('%v','%v')",
				ret, err,
				playbackData, nil)
		}
	}

	{
		ret, err := natsClient.Playback.Data("pb2")
		failed = err == nil || err.Error() != playbackErrorMessage
		if failed {
			t.Errorf("nc.Playback.Data('pb2') => ('%v','%v'), expected ('%v','%v')",
				ret, err,
				playbackData, playbackErrorMessage)
		}
	}

}
