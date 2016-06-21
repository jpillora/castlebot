package acmewrapper

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/xenolf/lego/acme"
)

// The ChallengeProvider fits the interface of lego/acme used for challenges.

// wrapperChallengeProvider is used to fit into the acme.ChallengeProvider interface,
// which allows us to use our own SNI code that is already being used to live-update
// our TLS certificates in the challenge process too
type wrapperChallengeProvider struct {
	w    *AcmeWrapper
	cert *tls.Certificate
}

// Present sets up the challenge domain thru SNI. Part of acme.ChallengeProvider interface
func (c *wrapperChallengeProvider) Present(domain, token, keyAuth string) error {
	logf("[acmewrapper] Started SNI server modification for %s", domain)
	// Use ACME's SNI challenge cert maker. How nice that it is exported :)
	cert, err := acme.TLSSNI01ChallengeCert(keyAuth)
	if err != nil {
		return err
	}

	// The returned cert has the info we want, but not parsed - so parse it
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return err
	}

	// The cert maker function gives us all the info we need in the cert itself
	for i := range cert.Leaf.DNSNames {

		// Add the cert to our AcmeWrapper. here, the names is the special SNI challenge domain
		// in the form "<Zi[0:32]>.<Zi[32:64]>.acme.invalid"
		c.w.AddSNI(cert.Leaf.DNSNames[i], &cert)
	}

	c.cert = &cert

	return nil

}

// CleanUp removes the challenge domain from SNI. Part of acme.ChallengeProvider interface
func (c *wrapperChallengeProvider) CleanUp(domain, token, keyAuth string) error {
	logf("[acmewrapper] End of SNI server modification for %s\n", domain)
	for i := range c.cert.Leaf.DNSNames {
		c.w.RemSNI(c.cert.Leaf.DNSNames[i])
	}
	return nil
}
