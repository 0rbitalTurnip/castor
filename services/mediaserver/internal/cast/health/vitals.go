package health

import (
	"fmt"
	"time"

	"github.com/stupside/castor/services/mediaserver/internal/cast/fetch"
	"github.com/stupside/castor/services/mediaserver/internal/media"
)

// Vitals is one reading of a watched cast: its measurements and terminal states.
type Vitals struct {
	// Landed is the bytes of artifact available now.
	Landed int64

	// sinceGrowth is how long no new media has arrived: Position when the producer states one, else Landed.
	sinceGrowth time.Duration

	// Position is the media made so far, and Speed that media per wall-clock second since the producer started.
	Position time.Duration
	Speed    media.Speed

	// Samples counts the speeds the producer stated, none before its first packet.
	Samples int

	// rate is media made per wall-clock second over the last rateSpan, which is what the producer delivers now.
	rate float64

	// sinceDeficit is how long the rate has been under playback.
	sinceDeficit time.Duration

	// ended is a producer that finished, which never stalls since its buffer will not grow again.
	ended  bool
	failed bool

	// overdue is a fact about the delivery's own patience, not about the producer.
	overdue bool

	// lead is how far a burn-in's transcription has committed, and leadDone whether it finished.
	subtitles bool
	lead      float64
	leadDone  bool

	// handed is the bytes the device took, and sinceFetch how long since one moved.
	handed     int64
	sinceFetch time.Duration

	// played is the media the device has played since it was handed the URL, less the time it was held.
	played time.Duration

	// holdable is a device the watch may pause; paused is one it holds, for sincePause.
	holdable   bool
	paused     bool
	sincePause time.Duration
}

func (h Vitals) String() string {
	return fmt.Sprintf("landed=%d position=%s rate=%.3gx speed=%.4gx samples=%d since_growth=%s handed=%d since_fetch=%s ahead=%s paused=%t",
		h.Landed, h.Position.Round(time.Second), h.rate, float64(h.Speed),
		h.Samples, h.sinceGrowth.Round(time.Second), h.handed, h.sinceFetch.Round(time.Second),
		h.ahead().Round(time.Second), h.paused)
}

// playable reports that there is something to hand a device.
func (h Vitals) playable() bool { return h.Landed > 0 }

// ahead is the least media the device provably still has to play: what was made, less what it has played.
func (h Vitals) ahead() time.Duration { return h.Position - h.played }

// paced is a producer delivering at least as fast as playback consumes.
func (h Vitals) paced() bool { return h.rate >= playbackRate }

// leads is a transcription far enough ahead, or finished.
func (h Vitals) leads() bool { return h.lead >= transcriptionLead.Seconds() || h.leadDone }

// cushioned is a buffer deep enough to ride out a source that goes quiet for a moment.
func (h Vitals) cushioned() bool { return h.Position >= readCushion }

const (
	// playbackRate is the pace a device plays at.
	playbackRate = 1.0

	// readCushion is the media a read holds before a device is started, since a device gives up on a stall long before one is judged.
	readCushion = 30 * time.Second

	// deliverBudget is how long a read may deliver under playback before the gate gives the link up.
	deliverBudget = fetch.BackoffMax

	// lowWater is the media left to a device under which the watch holds it; highWater is where it lets it go again.
	lowWater  = 10 * time.Second
	highWater = readCushion

	// pauseBudget is how long a device is held for a read to get ahead again before the cast is given up.
	pauseBudget = 2 * fetch.BackoffMax

	// transcriptionLead is how far a burn-in leads before playback starts.
	transcriptionLead = fetch.EncodeBurst + 10*time.Second

	// StallWindow outlasts two reconnect ceilings, so a reconnecting read is never called silent.
	StallWindow = 2*fetch.BackoffMax + 30*time.Second

	// fetchWindow is one reconnect ceiling: a device on the local network needs no longer.
	fetchWindow = fetch.BackoffMax

	// rateSpan is how far back the rate is measured: three segments of the longest common target duration.
	rateSpan = 30 * time.Second

	// minPace is the share of playback under which a producer counts as not growing.
	minPace = 0.5

	// pollInterval is under the producer's report period, so no sample is missed.
	pollInterval = 200 * time.Millisecond

	// reportInterval is how often a watch with nothing to decide says so.
	reportInterval = 5 * time.Second
)
