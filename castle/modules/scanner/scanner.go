package scanner

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/jpillora/icmpscan"
)

func New() *Scanner {
	s := &Scanner{}
	s.timer = time.NewTimer(time.Duration(0))
	s.timer.Stop()
	s.settings.Enabled = true
	s.settings.Interval = 5 * time.Minute
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
		Hosts icmpscan.Hosts `json:"hosts"`
	}
}

func (sc *Scanner) ID() string {
	return "scanner"
}

func (sc *Scanner) check() {
	b := backoff.Backoff{Max: 5 * time.Minute}
	for {
		//wait here for <interval>
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
	log.Printf("scan hosts")
	hosts, err := icmpscan.Run(icmpscan.Spec{})
	if err != nil {
		return err
	}
	log.Printf("found hosts %d", len(hosts))
	sc.results.Hosts = hosts
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
