package acmewrapper

import (
	"errors"
	"fmt"
	"time"

	"github.com/xenolf/lego/acme"
)

// http://stackoverflow.com/questions/15323767/does-golang-have-if-x-in-construct-similar-to-python
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// checks if the two arrays of strings contain the same elements
func arraysMatch(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, i := range a {
		if !stringInSlice(i, b) {
			return false
		}
	}
	return true
}

func getCertError(errmap map[string]error) error {
	if len(errmap) == 0 {
		return nil
	}

	// Check if the error is actually a TOS error (meaning we need
	// to agree to the TOS again)
	for _, err := range errmap {
		if _, ok := err.(acme.TOSError); ok {
			return err
		}
	}

	// Nope, return the full error
	return fmt.Errorf("%v", errmap)

}

// Renew generates a new certificate
func (w *AcmeWrapper) Renew() (err error) {
	if w.Config.RenewCallback != nil {
		w.Config.RenewCallback()
	}

	w.Lock()
	defer w.Unlock()

	if w.Config.AcmeDisabled {
		return errors.New("Can't renew cert when ACME is disabled")
	}

	// TODO: In the future, figure out how to get renewals working with
	// the information we have
	cert, errmap := w.client.ObtainCertificate(w.Config.Domains, true, nil)
	err = getCertError(errmap)

	if err != nil {
		// If it is not a TOS error, fail
		if _, ok := err.(acme.TOSError); !ok {
			return err
		}

		// There are new TOS

		// TODO: update registration with new TOS
		if !w.Config.TOSCallback(w.registration.TosURL) {
			return errors.New("Did not accept new TOS")
		}

		err = w.client.AgreeToTOS()
		if err != nil {
			return err
		}

		// We agreed to new TOS. try again
		cert, errmap = w.client.ObtainCertificate(w.Config.Domains, true, nil)
		err = getCertError(errmap)
		if err != nil {
			return err
		}
	}

	crt, err := tlsCert(cert)
	if err != nil {
		return err
	}

	// Write the certs to file if we are using file-backed stuff
	if w.Config.TLSCertFile != "" {
		w.writeCert(w.Config.TLSCertFile, w.Config.TLSKeyFile, cert)
	}

	w.certmutex.Lock()
	w.cert = crt
	w.certmutex.Unlock()
	return nil
}

// backgroundExpirationChecker is exactly that - it runs in the background
// and ensures that messages regarding certificate expiration as well as
// any renewals if ACME is configured are run on time.
func backgroundExpirationChecker(w *AcmeWrapper) {
	logf("[acmewrapper] Started background expiration checker\n")
	for {
		time.Sleep(w.Config.RenewCheck)
		logf("[acmewrapper] Checking if cert needs update...\n")
		if w.CertNeedsUpdate() {
			logf("[acmewrapper] ...yes it does\n")
			for {
				if !w.CertNeedsUpdate() {
					break
				}
				if !w.Config.AcmeDisabled {
					err := w.Renew()
					if err != nil && w.Config.RenewFailedCallback != nil {
						w.Config.RenewFailedCallback(err)
					}
					if err != nil {
						logf("[acmewrapper] Cert update renewal failed!")
					}
				}
				if !w.CertNeedsUpdate() {
					break
				}
				time.Sleep(w.Config.RetryDelay)
			}
		}

	}
}
