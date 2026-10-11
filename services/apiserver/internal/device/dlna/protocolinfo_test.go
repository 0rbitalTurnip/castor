package dlna

import (
	"slices"
	"testing"

	mediav1 "github.com/stupside/castor/gen/castor/media/v1"
)

func TestParseSinkProtocolInfo(t *testing.T) {
	avcSink := "http-get:*:audio/mpeg:*," +
		"http-get:*:video/mp2t:DLNA.ORG_PN=AVC_TS_HD_50_AC3_ISO," +
		"http-get:*:video/mp4:DLNA.ORG_PN=AVC_MP4_MP_HD_AAC"

	caps, _ := parseSinkProtocolInfo(avcSink)
	if got := codecs(caps.GetVideo()); !slices.Equal(got, []mediav1.VideoCodec{mediav1.VideoCodec_VIDEO_CODEC_H264}) {
		t.Errorf("AVC-only sink video = %v, want H.264 and no HEVC", got)
	}
	if got := codecs(caps.GetAudio()); !slices.Equal(got, []mediav1.AudioCodec{mediav1.AudioCodec_AUDIO_CODEC_AAC, mediav1.AudioCodec_AUDIO_CODEC_AC3}) {
		t.Errorf("sink audio = %v, want AAC and AC-3 from the AV profiles, no E-AC-3", got)
	}
	if got := caps.GetContainers(); !slices.Equal(got, []mediav1.Container{mediav1.Container_CONTAINER_MPEGTS, mediav1.Container_CONTAINER_MP4}) {
		t.Errorf("sink containers = %v, want MPEG-TS and MP4", got)
	}
	withHEVC, _ := parseSinkProtocolInfo(avcSink + ",http-get:*:video/mp2t:DLNA.ORG_PN=HEVC_TS_MAIN_HD")
	if got := codecs(withHEVC.GetVideo()); !slices.Contains(got, mediav1.VideoCodec_VIDEO_CODEC_HEVC) {
		t.Errorf("HEVC_TS sink video = %v, want HEVC advertised", got)
	}
	// No video codec makes negotiateCaps refuse the device.
	if got, _ := parseSinkProtocolInfo("garbage,http-get:*:audio/mpeg:*"); len(got.Video) != 0 {
		t.Errorf("unusable sink should yield no video, got %v", got.Video)
	}
}

func TestAContainerIsSpokenForByTheEntryNamingACodec(t *testing.T) {
	caps, sinks := parseSinkProtocolInfo("http-get:*:video/mpeg:*,http-get:*:video/mp2t:DLNA.ORG_PN=MPEG_TS_SD_EU,http-get:*:video/mp2t:DLNA.ORG_PN=AVC_TS_HD_50_AC3_ISO")
	if got := sinks[mediav1.Container_CONTAINER_MPEGTS].features; got != "DLNA.ORG_PN=AVC_TS_HD_50_AC3_ISO" {
		t.Errorf("MPEG-TS entry = %q, want the one naming a codec", got)
	}
	if got := caps.GetContainers(); !slices.Equal(got, []mediav1.Container{mediav1.Container_CONTAINER_MPEGTS}) {
		t.Errorf("containers = %v, want MPEG-TS alone: video/mpeg is MPEG-PS", got)
	}
	if caps.GetServedContainer() != mediav1.Container_CONTAINER_MPEGTS {
		t.Errorf("served container = %v, want the first the device accepts", caps.GetServedContainer())
	}
}
