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
	SeenAt time.Time `json:"seenAt"`
}

func New() *Scanner {
	s := &Scanner{}
	s.timer = time.NewTimer(time.Duration(0))
	s.timer.Stop()
	s.settings.Enabled = true
	s.settings.Interval = 2 * time.Minute
	s.results.Hosts = map[string]host{}
	go s.check()
	return s
}

type Scanner struct {
	updates  chan interface{}
	timer    *time.Timer
	settings struct {
		Enabled  bool
		Interval time.Duration
	}
	results struct {
		sync.Mutex
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
	hosts, err := icmpscan.Run(icmpscan.Spec{
		Hostnames: true,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		return err
	}
	now := time.Now()
	for _, ih := range hosts {
		ip := ih.IP.String()
		h, ok := sc.results.Hosts[ip]
		if !ok {
			h.Host = ih
		}
		h.Active = ih.Active
		if ih.Active {
			if h.SeenAt.IsZero() {
				log.Printf("[scanner] found host: %s", ih.IP)
			}
			h.SeenAt = now
			h.Error = ih.Error
		}
		if ih.Hostname != "" {
			h.Hostname = ih.Hostname
		}
		if ih.RTT > 0 {
			h.RTT = ih.RTT
		}
		sc.results.Hosts[ip] = h
	}
	sc.results.ScannedAt = now
	sc.updates <- &sc.results
	return nil
}

func (sc *Scanner) Status(updates chan interface{}) {
	sc.updates = updates
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
