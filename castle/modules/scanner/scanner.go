package scanner

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/jpillora/icmpscan"
)

type host struct {
	IP       net.IP        `json:"ip"`
	MAC      string        `json:"mac,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	RTT      time.Duration `json:"rtt,omitempty"`
	SeenAt   time.Time     `json:"seenAt"`
	ActiveAt time.Time     `json:"activeAt"`
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
		//ip and mac as strings
		mac := ih.MAC
		ip := ih.IP.String()
		//decide on key
		key := mac
		if key == "" {
			key = ip
		}
		h := sc.results.Hosts[key]
		h.IP = ih.IP
		if mac != "" {
			h.MAC = mac
		}
		if ih.Hostname != "" {
			h.Hostname = ih.Hostname
		}
		if ih.RTT > 0 {
			h.RTT = ih.RTT
		}
		//mac key, wipe ip only entry
		if key == mac {
			if h2, ok := sc.results.Hosts[ip]; ok && h2.MAC == "" {
				delete(sc.results.Hosts, ip)
			}
		}
		//calculate seen
		if h.SeenAt.IsZero() {
			log.Printf("[scanner] found host: %s", ih.IP)
		}
		if now.Sub(h.SeenAt) > sc.settings.ActiveAtThreshold {
			h.ActiveAt = now
		}
		h.SeenAt = now
		//update host
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
