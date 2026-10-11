package execute

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stupside/castor/services/mediaserver/internal/cast/deliver"
	"github.com/stupside/castor/services/mediaserver/internal/cast/recovery"
	"github.com/stupside/castor/services/mediaserver/internal/ffmpeg"
	"github.com/stupside/castor/services/mediaserver/internal/media"
	"github.com/stupside/castor/services/mediaserver/internal/probe"
)

// TestTheReadsFloorEncodeIsCappedAtTheCastsCeiling: an uncapped read encode falls below realtime.
func TestTheReadsFloorEncodeIsCappedAtTheCastsCeiling(t *testing.T) {
	ffmpegPath, ffprobePath := requireFFmpegTools(t)
	origin := serveFixture(t, ffmpegPath)

	// The fixture is 240p; the ceiling sits under it.
	c := realCast(&fakeDevice{}, ffmpegPath, ffprobePath)
	c.MaxHeight = 120
	program := programFromStream(t, origin.stream())

	s := readingPipeline(t, c, recovery.Attempt{Program: program, Fetch: sourceFetchPlan(t, program, testReadDeadline), Decode: media.Axes{Video: true}})
	followed, err := s.follow(program)
	if err != nil {
		t.Fatal(err)
	}
	buf, err := s.read(t.TempDir(), followed)
	if err != nil {
		t.Fatal(err)
	}
	<-buf.reader.Done()
	if err := buf.reader.Err(); err != nil {
		t.Fatalf("the read failed: %v\n%q", err, buf.reader.Evidence())
	}

	info, _, err := probe.FFprobe(ffprobePath).File(buf.reader.spool.Path()).Probe(t.Context())
	if err != nil {
		t.Fatalf("probing the buffer this read wrote: %v", err)
	}
	if info.VideoHeight != 120 {
		t.Errorf("the read produced a %dp picture, want the 120p ceiling", info.VideoHeight)
	}
}

// A spool that stops taking writes ends the read at once instead of leaving its ffmpeg blocked on a full pipe.
func TestAFailedSpoolWriteKillsTheReader(t *testing.T) {
	sp, err := deliver.NewSpool(filepath.Join(t.TempDir(), "spool.ts"))
	if err != nil {
		t.Fatal(err)
	}
	sp.CloseWrite(nil)

	// yes writes to its stdout until it is killed.
	proc, err := ffmpeg.Start(t.Context(), "yes", ffmpeg.Command{}, ffmpeg.Options{})
	if err != nil {
		t.Fatal(err)
	}
	p := &pull{proc: proc, spool: sp, done: make(chan struct{})}
	go p.run(t.Context())

	select {
	case <-p.Done():
	case <-time.After(10 * time.Second):
		proc.Kill()
		t.Fatal("the read never ended after its spool refused a write")
	}
	if p.Err() == nil {
		t.Error("the read reported success after its spool refused a write")
	}
}
