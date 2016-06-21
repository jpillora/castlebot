package acmewrapper

import (
	"time"

	"github.com/xenolf/lego/acme"
)

const (
	// The server to use by default
	DefaultServer = "https://acme-v01.api.letsencrypt.org/directory"
	// Default type for the private key
	DefaultKeyType = acme.RSA2048

	// The default port to use for initializing certs on startup
	DefaultAddress = ":443"

	// DefaultRenewTime is the time period before cert expiration to attempt renewal
	DefaultRenewTime  = 30 * 24 * time.Hour
	DefaultRetryDelay = 1 * 24 * time.Hour // Retry once a day
	DefaultRenewCheck = 12 * time.Hour     // The time between checks for renewal
)

// TOSAgree always agrees to the terms of service. This should only be really used if
// you realize that you could be selling your soul without being notified.
func TOSAgree(agreementURL string) bool {
	return true
}

// TOSDecline always declines to the terms of service. This can be usd for testing, when you want
// to make sure that ACME is really off, or that the user is being loaded.
func TOSDecline(agreementURL string) bool {
	return false
}

// TOSCallback is a callback to run when the TOS have changed, and need to be agreed to.
// The returned bool is agree/not agree
type TOSCallback func(agreementURL string) bool

// Config is the setup to use for generating your TLS keys.
// While the only required component is Server, it is recommended that you save at least your
// TLS cert and key
type Config struct {
	// The ACME server to query for key/cert. Default (Let's Encrypt) used if not set
	Server string

	// The domain for which to generate your certificate. Suppose you own mysite.com.
	// The domains to pass in are Domains: []string{"mysite.com","www.mysite.com"}. Don't
	// forget about the www version of your domain.
	Domains []string

	// The file to read/write the private key from/to. If this is not empty, and the file does not exist,
	// then the user is assumed not to be registered, and the file is created. if this is empty, then
	// a new private key is generated and used for all queries. The private key is lost on stopping the program.
	PrivateKeyFile string
	PrivateKeyType acme.KeyType // The private key type. Default is 2048 (RSA)

	// The file to read/write registration info to. The ACME protocol requires remembering some details
	// about a registration. Therefore, the file is saved at the given location.
	// If not given, and PrivateKeyFile is given, then gives an error - if you're saving your private key,
	// you need to save your user registration.
	RegistrationFile string

	Email string `json:"email"` // Optional user email

	// File names at which to read/write the TLS key and certificate. These are optional. If there
	// is no file given, then the keys are kept in memory. NOTE: You need write access to these files,
	// since they are overwritten each time a new certificate is requested.
	// Also, it is HIGHLY recommended that you save the files, since Let's Encrypt has fairly low limits
	// for how often certs for the same site can be requested (5/week at the time of writing).
	TLSCertFile string
	TLSKeyFile  string

	RenewTime  time.Duration // The time in seconds until expiration of current cert that renew is attempted. If not set, default is 30d
	RetryDelay time.Duration // The time in seconds to delay between attempts at renewing if renewal fails. (1 day)
	RenewCheck time.Duration // The time inbetween checks for renewal. Default is 12h

	// The callback to use prompting the user to agree to the terms of service. A special Agree is built in, so
	// you can set TOSCallback: TOSAgree
	TOSCallback TOSCallback

	// If there is no certificate set up at all, we need to generate an inital one
	// to jump-start the server. Therefore, you should input the port that you
	// will use when running listen. If there are no certs, it runs a temporary mini
	// server at that location to generate initial certificates. Once that is done,
	// all further renewals are done through the SNI interface to your own server code.
	// The default here is 443
	Address string

	// This callback is run before each attempt at renewing. If not set, it simply isn't run.
	RenewCallback func()
	// RenewFailedCallback is run if renewing failed.
	RenewFailedCallback func(error)

	// When this is set to True, no ACME-related things happen - it just passes through your
	// key and cert directly.
	AcmeDisabled bool

	// When this callback is defined, it will be used to save all files.
	// If this callback returns acmewrapper.ErrNotHandled, it will fallback to save file to disk.
	SaveFileCallback func(path string, contents []byte) error
	// When this callback is defined, it will be used to load all files.
	// If this callback does not find the file at the provided path, it must return os.ErrNotExist.
	// If this callback returns acmewrapper.ErrNotHandled, it will fallback to load file from disk.
	LoadFileCallback func(path string) (contents []byte, err error)
}
