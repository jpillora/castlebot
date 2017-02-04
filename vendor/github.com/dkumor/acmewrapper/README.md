# ACMEWrapper

Add Let's Encrypt support to your golang server in 10 lines of code.

[![GoDoc](https://godoc.org/github.com/dkumor/acmewrapper?status.svg)](https://godoc.org/github.com/dkumor/acmewrapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/dkumor/acmewrapper)](https://goreportcard.com/report/github.com/dkumor/acmewrapper)
[![Build Status](https://travis-ci.org/dkumor/acmewrapper.svg?branch=master)](https://travis-ci.org/dkumor/acmewrapper)

```go
w, err := acmewrapper.New(acmewrapper.Config{
	Domains: []string{"example.com","www.example.com"},
	Address: ":443",

	TLSCertFile: "cert.pem",
	TLSKeyFile:  "key.pem",

	// Let's Encrypt stuff
	RegistrationFile: "user.reg",
	PrivateKeyFile:   "user.pem",

	TOSCallback: acmewrapper.TOSAgree,
})


if err!=nil {
	log.Fatal("acmewrapper: ", err)
}

listener, err := tls.Listen("tcp", ":443", w.TLSConfig())
```

Acmewrapper is built upon https://github.com/xenolf/lego, and handles all certificate generation, renewal
and replacement automatically. After the above code snippet, your certificate will automatically be renewed 30 days before expiring without downtime. Any files that don't exist will be created, and your "cert.pem" and "key.pem" will be kept up to date.

Since Let's Encrypt is usually an option that can be turned off, the wrapper allows disabling ACME support and just using normal certificates, with the bonus of allowing live reload (ie: change your certificates during runtime).

And finally, *technically*, none of the file names shown above are actually necessary. The only needed fields are Domains and TOSCallback. Without the given file names, acmewrapper runs in-memory. Beware, though: if you do that, you might run into rate limiting from Let's Encrypt if you restart too often!

## How It Works

Let's Encrypt has SNI support for domain validation. That means we can update our certificate if we control the TLS configuration of a server. That is exactly what acmewrapper does. Not only does it transparently update your server's certificate, but it uses its control of SNI to pass validation tests.

This means that *no other changes* are needed to your code. You don't need any special handlers or hidden directories. So long as acmewrapper is able to set your TLS configuration, and your TLS server is running on port 443, you can instantly have a working Let's Encrypt certificate.

## Notes

Currently, Go 1.4 and above are supported

- There was a breaking change on 3/20/16: all time periods in the configuration were switched from int64 to time.Duration.

## Example

You can go into `./example` to find a sample basic http server that will serve a given folder over https with Let's Encrypt.

Another simple example is given below:

### Old Code

This is sample code before adding Let's Encrypt support:

```go
package main

import (
    "io"
    "net/http"
    "log"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "hello, world!\n")
}

func main() {
    http.HandleFunc("/hello", HelloServer)
    err := http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```

### New Code

Adding let's encrypt support is a matter of setting the tls config:

```go
package main

import (
    "io"
    "net/http"
    "log"
	"crypto/tls"

	"github.com/dkumor/acmewrapper"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "hello, world!\n")
}

func main() {
	mux := http.NewServeMux()
    mux.HandleFunc("/hello", HelloServer)

	w, err := acmewrapper.New(acmewrapper.Config{
		Domains: []string{"example.com","www.example.com"},
		Address: ":443",

		TLSCertFile: "cert.pem",
		TLSKeyFile:  "key.pem",

		RegistrationFile: "user.reg",
		PrivateKeyFile:   "user.pem",

		TOSCallback: acmewrapper.TOSAgree,
	})


	if err!=nil {
		log.Fatal("acmewrapper: ", err)
	}

	tlsconfig := w.TLSConfig()

	listener, err := tls.Listen("tcp", ":443", tlsconfig)
    if err != nil {
        log.Fatal("Listener: ", err)
    }

	// To enable http2, we need http.Server to have reference to tlsconfig
	// https://github.com/golang/go/issues/14374
	server := &http.Server{
		Addr: ":443",
		Handler:   mux,
		TLSConfig: tlsconfig,
	}
	server.Serve(listener)
}
```

## Custom File Handlers

While ACMEWrapper saves certificates to the filesystem by default, you can save all relevant files in your database by overloading the read and write functions

```go
w, err := acmewrapper.New(acmewrapper.Config{
		Domains: []string{"example.com","www.example.com"},
		Address: ":443",

		TLSCertFile: "CERTIFICATE",
		TLSKeyFile:  "TLSKEY",

		RegistrationFile: "REGISTRATION",
		PrivateKeyFile:   "PRIVATEKEY",

		TOSCallback: acmewrapper.TOSAgree,

		SaveFileCallback: func(path string, contents []byte) error {
			// the path is the file name as set up in the configuration - the certificate will be "CERTIFICATE", etc.
		},
		// If this callback does not find the file at the provided path, it must return os.ErrNotExist.
		// If this callback returns acmewrapper.ErrNotHandled, it will fallback to load file from disk.
		LoadFileCallback func(path string) (contents []byte, err error) {
			return os.ErrNotExist
		},
	})

```

## Testing

Running the tests is a bit of a chore, since it requires a valid domain name, and access to port 443.
This is because ACMEWrapper uses the Let's Encrypt staging server to make sure the code is working.

To test on your own server, you need to change the domain name to your domain, and set a custom testing port
that will be routed to 443:

```bash
go test -c
sudo setcap cap_net_bind_service=+ep acmewrapper.test
export TLSADDRESS=":443"
export DOMAIN_NAME="example.com"
./acmewrapper.test
```
