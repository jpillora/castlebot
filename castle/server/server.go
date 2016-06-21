package server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/dkumor/acmewrapper"
)

func New(db *bolt.DB, root http.Handler, defaultPort int) *Server {
	s := &Server{
		running: make(chan error),
		adb:     &acmeDB{DB: db},
		root:    root,
	}
	s.Config.HTTP.Port = defaultPort
	return s
}

type Server struct {
	Config struct {
		HTTP struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"http"`
		HTTPS struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Hostname string `json:"hostname"`
			Email    string `json:"email"`
		} `json:"https"`
	}
	updated                     bool
	running                     chan error
	adb                         *acmeDB
	root                        http.Handler
	listenMut                   sync.Mutex
	httpListener, httpsListener net.Listener
}

func (s *Server) listen() error {
	//only 1 listener at a time
	s.listenMut.Lock()
	defer s.listenMut.Unlock()
	s.updated = false
	//setup tls/tcp listener
	s.httpsListener = nil
	if s.Config.HTTPS.Port > 0 && s.Config.HTTPS.Hostname != "" {
		w, err := acmewrapper.New(acmewrapper.Config{
			Domains:          []string{s.Config.HTTPS.Hostname},
			Address:          fmt.Sprintf(":%d", s.Config.HTTPS.Port),
			TLSCertFile:      "cert.pem",
			TLSKeyFile:       "key.pem",
			RegistrationFile: "user.reg",
			PrivateKeyFile:   "user.pem",
			Email:            s.Config.HTTPS.Email,
			TOSCallback:      acmewrapper.TOSAgree,
			SaveFileCallback: s.adb.SaveFileCallback,
			LoadFileCallback: s.adb.LoadFileCallback,
		})
		if err != nil {
			return err
		}
		addr := fmt.Sprintf("%s:%d", s.Config.HTTPS.Host, s.Config.HTTPS.Port)
		l, err := tls.Listen("tcp", addr, w.TLSConfig())
		if err != nil {
			return err
		}
		log.Printf("Listening on https://%s:%d", s.Config.HTTPS.Hostname, s.Config.HTTPS.Port)
		s.httpsListener = l
	}
	s.httpListener = nil
	if s.Config.HTTP.Port > 0 {
		addr := fmt.Sprintf("%s:%d", s.Config.HTTP.Host, s.Config.HTTP.Port)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Printf("Listening on http://%s", addr)
		s.httpListener = l
	}
	if s.httpsListener == nil && s.httpListener == nil {
		return errors.New("no http listeners defined")
	}
	wg := sync.WaitGroup{}
	if s.httpsListener != nil {
		wg.Add(1)
		go func() {
			if err := http.Serve(s.httpsListener, s.root); err != nil {
				log.Printf("https listener: %s", err)
			}
			wg.Done()
		}()
	}
	if s.httpListener != nil {
		wg.Add(1)
		go func() {
			if err := http.Serve(s.httpListener, s.root); err != nil {
				log.Printf("http listener: %s", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}

func (s *Server) Get() interface{} {
	return &s.Config
}

func (s *Server) Set(j json.RawMessage) error {
	if j != nil {
		if err := json.Unmarshal(j, &s.Config); err != nil {
			return err
		}
	}
	//defaults
	if s.Config.HTTP.Port == 0 {
		s.Config.HTTP.Port = 4000
	}
	if s.Config.HTTP.Host == "" {
		s.Config.HTTP.Host = "0.0.0.0"
	}
	if s.Config.HTTPS.Host == "" {
		s.Config.HTTPS.Host = "0.0.0.0"
	}
	//close existing listeners
	s.updated = true
	if s.httpsListener != nil {
		if err := s.httpsListener.Close(); err != nil {
			return err
		}
	}
	if s.httpListener != nil {
		if err := s.httpListener.Close(); err != nil {
			return err
		}
	}
	//listen
	go func() {
		err := s.listen()
		if !s.updated {
			s.running <- err
			return
		}
	}()
	return nil
}

func (s *Server) Wait() error {
	return <-s.running
}
