package acmewrapper

import (
	"crypto"
	"crypto/tls"
	"log"
	"os"
	"sync"

	"github.com/xenolf/lego/acme"
)

type LoggerInterface interface {
	Printf(format string, v ...interface{})
}

// Allows to use a custom logger for logging purposes
var Logger LoggerInterface

func logf(s string, v ...interface{}) {
	if Logger == nil {
		log.Printf(s, v...)
	} else {
		Logger.Printf(s, v...)
	}
}

// AcmeWrapper is the main object which controls tls certificates and their renewals
type AcmeWrapper struct {
	sync.Mutex // configmutex ensures that settings for the ACME stuff don't happen in parallel
	Config     Config

	certmutex sync.RWMutex // certmutex is used to make sure that replacing certificates doesn't asplode

	// Our user's private key & registration. Both are needed in order to be able
	// to renew/generate new certs.
	privatekey   crypto.PrivateKey
	registration *acme.RegistrationResource

	// A map of custom certificates associated with special SNIs. The SNI request
	// passes through here
	certs map[string]*tls.Certificate

	// The current TLS cert used for SSL requests when the SNI doesn't match the map
	cert *tls.Certificate

	// The ACME client
	client *acme.Client
}

// GetEmail returns the user email (if any)
// NOTE: NOT threadsafe
func (w *AcmeWrapper) GetEmail() string {
	return w.Config.Email
}

// GetRegistration returns the registration currently being used
// NOTE: NOT threadsafe
func (w *AcmeWrapper) GetRegistration() *acme.RegistrationResource {
	return w.registration
}

// GetPrivateKey returns the private key for the given user.
// NOTE: NOT threadsafe
func (w *AcmeWrapper) GetPrivateKey() crypto.PrivateKey {
	return w.privatekey
}

// GetCertificate returns the current TLS certificate
func (w *AcmeWrapper) GetCertificate() *tls.Certificate {
	w.certmutex.RLock()
	defer w.certmutex.RUnlock()
	return w.cert
}

// AddSNI adds a domain name and certificate pair to the AcmeWrapper.
// Whenever a request is for the passed domain, its associated certifcate is returned.
func (w *AcmeWrapper) AddSNI(domain string, cert *tls.Certificate) {
	w.certmutex.Lock()
	defer w.certmutex.Unlock()

	w.certs[domain] = cert
}

// RemSNI removes a domain name and certificate pair from the AcmeWrapper. It is assumed that
// they were added using AddSNI.
func (w *AcmeWrapper) RemSNI(domain string) {
	w.certmutex.Lock()
	defer w.certmutex.Unlock()

	delete(w.certs, domain)
}

// TLSConfigGetCertificate is the main function used in the ACME wrapper. This is set in tls.Config to
// the GetCertificate property. Note that Certificates must be empty for it to be called
// correctly, so unless you know what you're doing, just use AcmeWrapper.TLSConfig()
func (w *AcmeWrapper) TLSConfigGetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	w.certmutex.RLock()
	defer w.certmutex.RUnlock()

	// If the SNI is in the certs map, return that cert
	if _, ok := w.certs[clientHello.ServerName]; ok {
		return w.certs[clientHello.ServerName], nil
	}

	// Otherwise, return the default cert
	return w.cert, nil
}

// TLSConfig returns a TLS configuration that will automatically work with the golang ssl listener.
// This sets it up so that the server automatically uses a working cert, and updates the cert when
// necessary.
func (w *AcmeWrapper) TLSConfig() *tls.Config {
	return &tls.Config{
		//Go 1.6 "allows Listen to succeed when the Config has a nil Certificates, as long as the
		//GetCertificate callback is set" See https://golang.org/doc/go1.6#minor_library_changes.
		//So to add Go 1.5 support, we provide a certificate that will never be used.
		// This must be a valid cert, since if the client does not have SNI enabled in go1.4,
		// then there could be issues. AcmeWrapper does not support clients without SNI enabled at this time.
		Certificates:   []tls.Certificate{*w.cert},
		GetCertificate: w.TLSConfigGetCertificate,
	}
}

// AcmeDisabled allows to enable/disable acme-based certificate. Note that it is assumed that
// this function is only called during server runtime (ie, your server is already listening).
// its main purpose is to enable live reload of acme configuration. Do NOT set AcmeDisabled
// in AcmeWrapper.Config, since it will panic.
func (w *AcmeWrapper) AcmeDisabled(set bool) error {
	w.Lock()
	w.Config.AcmeDisabled = set
	w.Unlock()
	if !set && w.client == nil {
		return w.initACME(true)
	}
	return nil
}

// SetNewCert loads a new TLS key/cert from the given files. Running it with the same
// filenames as existing cert will reload them
func (w *AcmeWrapper) SetNewCert(certfile, keyfile string) error {
	cert, err := w.loadX509KeyPair(certfile, keyfile)
	if err != nil {
		return err
	}
	w.certmutex.Lock()
	w.cert = &cert
	w.certmutex.Unlock()

	return nil
}

// New generates an AcmeWrapper given a configuration
func New(c Config) (*AcmeWrapper, error) {
	var err error
	// First, set up the default values for any settings that require
	// values
	if c.Server == "" {
		c.Server = DefaultServer
	}
	if c.PrivateKeyType == "" {
		c.PrivateKeyType = DefaultKeyType
	}
	if c.RenewTime == 0 {
		c.RenewTime = DefaultRenewTime
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}
	if c.RenewCheck == 0 {
		c.RenewCheck = DefaultRenewCheck
	}
	if c.Address == "" {
		c.Address = DefaultAddress
	}

	// Now set up the actual wrapper

	var w AcmeWrapper
	w.Config = c
	w.certs = make(map[string]*tls.Certificate)

	// Now load up the key and cert files for TLS if they are set
	if c.TLSKeyFile != "" || c.TLSCertFile != "" {
		err = w.SetNewCert(c.TLSCertFile, c.TLSKeyFile)
		if err != nil {
			if !os.IsNotExist(err) || c.AcmeDisabled {
				// The TLS key and cert file are only
				// allowed to not be there if ACME will generate them
				// TODO: We don't check here if both are missing vs 1 missing
				return nil, err
			}

		}
	}

	// If acme is enabled, initialize it!
	if !c.AcmeDisabled {
		// Initialize the ACME user
		// initUser succeeding initializes:
		//	- w.privatekey
		//	- w.registration
		//	- w.client
		if err = w.initACME(false); err != nil {
			return nil, err
		}
	}

	// Finally, start the background routine
	go backgroundExpirationChecker(&w)

	return &w, nil
}
