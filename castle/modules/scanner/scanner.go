package scanner

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/jpillora/icmpscan"
)

type host struct {
	*icmpscan.Host
	SeenAt   time.Time `json:"seenAt"`
	ActiveAt time.Time `json:"activeAt"`
}

func New() *Scanner {
	s := &Scanner{}
	s.timer = time.NewTimer(time.Duration(0))
	s.timer.Stop()
	s.settings.Enabled = true
	s.settings.Interval = 2 * time.Minute
	s.settings.ActiveAtThreshold = 15 * time.Minute
	s.results.Hosts = map[string]host{}
	go s.check()
	return s
}

type Scanner struct {
	updates  chan interface{}
	timer    *time.Timer
	settings struct {
		Enabled           bool
		Debug             bool
		Interval          time.Duration
		ActiveAtThreshold time.Duration
	}
	results struct {
		sync.Mutex
		Scanning  bool            `json:"scanning"`
		ScannedAt time.Time       `json:"scannedAt"`
		Hosts     map[string]host `json:"hosts"`
	}
}

func (sc *Scanner) ID() string {
	return "scanner"
}

func (sc *Scanner) check() {
	b := backoff.Backoff{Max: 5 * time.Minute}
	for {
		//wait here for <interval>
		//short-circuited by Set()
		sc.timer.Reset(sc.settings.Interval)
		log.Printf("[scanner] wait")
		<-sc.timer.C
		//scan!
		if err := sc.scan(); err != nil {
			log.Printf("[scanner] failed: %s", err)
			time.Sleep(b.Duration())
		} else {
			b.Reset()
		}
	}
}

func (sc *Scanner) scan() error {
	if !sc.settings.Enabled {
		return nil
	}
	//show scan state
	sc.results.Scanning = true
	sc.push()
	defer func() {
		sc.results.Scanning = false
		sc.push()
	}()
	//perform scan
	log.Printf("[scanner] start")
	hosts, err := icmpscan.Run(icmpscan.Spec{
		MACs:      true,
		Hostnames: true,
		Timeout:   5 * time.Second,
		Log:       sc.settings.Debug,
	})
	if err != nil {
		return err
	}
	now := time.Now()
	for _, ih := range hosts {
		key := ih.MAC
		if key == "" {
			key = ih.IP.String()
		}
		h, ok := sc.results.Hosts[key]
		if !ok {
			h.Host = ih
		}
		if h.SeenAt.IsZero() {
			log.Printf("[scanner] found host: %s", ih.IP)
		}
		if now.Sub(h.SeenAt) > sc.settings.ActiveAtThreshold {
			h.ActiveAt = now
		}
		h.SeenAt = now
		if ih.Hostname != "" {
			h.Hostname = ih.Hostname
		}
		if ih.MAC != "" {
			h.MAC = ih.MAC
		}
		if ih.RTT > 0 {
			h.RTT = ih.RTT
		}
		sc.results.Hosts[key] = h
	}
	sc.results.ScannedAt = now
	sc.push()
	return nil
}

func (sc *Scanner) Status(updates chan interface{}) {
	sc.updates = updates
	sc.push()
}

func (sc *Scanner) push() {
	if sc.updates != nil {
		sc.updates <- &sc.results
	}
}

func (sc *Scanner) Get() interface{} {
	return &sc.settings
}

func (sc *Scanner) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &sc.settings); err != nil {
			return err
		}
	}
	sc.timer.Reset(0)
	return nil
}
