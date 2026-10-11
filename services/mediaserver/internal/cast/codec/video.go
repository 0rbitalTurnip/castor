package codec

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/stupside/castor/services/mediaserver/internal/cast/container"
	"github.com/stupside/castor/services/mediaserver/internal/ffmpeg"
	"github.com/stupside/castor/services/mediaserver/internal/media"
)

var videoCarriageRefusals = []refusalRule{{
	// Container carriage asks whether ffmpeg's muxer has a stream type for the codec.
	reason: reasonContainerVideo,
	when:   func(in Inputs) bool { return container.Uncarried(in.Probe, in.Into).Video },
}, {
	// One encoder gives the device one stream: a copy keeps each piece's own size and timestamps.
	reason: reasonSpliced,
	when:   func(in Inputs) bool { return in.Spliced },
}, {
	// ffmpeg turns the frames when it decodes them, so only a re-encode lands upright.
	reason: reasonRotated,
	when:   func(in Inputs) bool { return in.Probe.VideoRotation != 0 },
}, {
	// Kept apart from container's refusal because they are different failures with different lifetimes.
	reason: reasonVideoCopyFailed,
	when:   func(in Inputs) bool { return in.Decode.Video },
}}

var videoRefusals = slices.Concat([]refusalRule{{
	// The device's advertised envelope is the only thing that makes a copy safe at all.
	reason: reasonDeviceVideo,
	when:   func(in Inputs) bool { return !in.Caps.CanCopyVideo(in.Probe) },
}}, videoCarriageRefusals, []refusalRule{{
	// The ceiling is the user's request and may not be bypassed.
	reason: reasonHeightLimit,
	when:   func(in Inputs) bool { return !in.MaxHeight.Admits(in.Probe.VideoHeight) },
}, {
	reason: reasonInterlaced,
	when:   func(in Inputs) bool { return in.Probe.VideoInterlaced && !in.Caps.Deinterlaces },
}, {
	reason: reasonHDRPolicy,
	when:   func(in Inputs) bool { return in.Probe.VideoHDR },
}, {
	// A cue drawn into the picture needs decoded frames; a copy has no frames to draw on.
	reason: reasonSubtitleBurnIn,
	when:   func(in Inputs) bool { return in.BurnIn != "" },
}})

func decideVideo(ctx context.Context, in Inputs) (Track[VideoEncode], []Reason, error) {
	if in.Measured && in.Probe.VideoCodec == "" {
		return CopyVideo(), nil, nil
	}
	refused := refuse(videoRefusals, in)
	if len(refused) == 0 {
		return CopyVideo(), nil, nil
	}
	host := func(c media.Codec) (ffmpeg.Encoder, bool) { return in.Encoders(ctx, c) }
	enc, t, ok := selectVideoEncoder(in.Caps, host)
	if !ok {
		return Track[VideoEncode]{}, refused, noVideoEncoder(in.Caps)
	}
	return Encode(VideoEncode{
		Encoder:             enc,
		Deinterlace:         in.Probe.VideoInterlaced && !in.Caps.Deinterlaces,
		ToneMap:             in.Probe.VideoHDR,
		MaxFrameRate:        playbackFrameRate,
		Bitrate:             t.bitrate,
		Maxrate:             t.maxrate,
		Bufsize:             t.bufsize,
		MaxHeight:           in.MaxHeight,
		KeyframeIntervalSec: keyframeSeconds,
		SubtitleTextFile:    in.BurnIn,
	}), refused, nil
}

// playbackFrameRate is the most any device castor serves shows in a second; above it a re-encode drops frames.
const playbackFrameRate = 60

func selectVideoEncoder(caps media.Capabilities, selectEncoder func(media.Codec) (ffmpeg.Encoder, bool)) (ffmpeg.Encoder, videoTarget, bool) {
	for _, t := range videoTargets {
		if !caps.SupportsVideoCodec(t.codec) {
			continue
		}
		if enc, ok := selectEncoder(t.codec); ok && holdsRealtime(enc, t.codec) {
			return enc, t, true
		}
	}
	// No common strategy is safer than inventing support the device never advertised.
	return ffmpeg.Encoder{}, videoTarget{}, false
}

func holdsRealtime(enc ffmpeg.Encoder, codec media.Codec) bool {
	return enc.Hardware || codec == media.CodecH264
}

func noVideoEncoder(caps media.Capabilities) error {
	advertised := make([]string, 0, len(caps.Video))
	for _, s := range caps.Video {
		if !slices.Contains(advertised, string(s.Codec)) {
			advertised = append(advertised, string(s.Codec))
		}
	}
	return fmt.Errorf("no video encoder for this device: it decodes %s", cmp.Or(strings.Join(advertised, ", "), "no video codec"))
}

type videoTarget struct {
	codec                     media.Codec
	bitrate, maxrate, bufsize string
}

// floorVideo is the codec every device decodes, and the one a read-once buffer is produced in.
var floorVideo = videoTarget{codec: media.CodecH264, bitrate: "4M", maxrate: "4M", bufsize: "8M"}

// videoTargets is every codec castor encodes video to, in preference order.
var videoTargets = []videoTarget{
	{codec: media.CodecHEVC, bitrate: "2M", maxrate: "2M", bufsize: "4M"},
	floorVideo,
}

const keyframeSeconds = 2

const floorVideoQuality = 23
