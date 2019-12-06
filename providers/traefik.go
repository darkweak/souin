package providers

import (
	"log"
	"github.com/fsnotify/fsnotify"
	"fmt"
	"io/ioutil"
	"github.com/darkweak/souin/providers/traefik"
	"encoding/json"
	"encoding/base64"
	"strings"
	"crypto/tls"
)

const FILE_LOCATION = "/app/src/github.com/darkweak/souin/acme.json"

func loadFromConfigFile(certificatesProviders *CommonProvider, tlsconfig *tls.Config, configChannel *chan int) {
	acmeFile, err := ioutil.ReadFile(FILE_LOCATION)
	if nil != err {
		panic(err)
	}

	certificates := &traefik.AcmeFile{}
	err = json.Unmarshal([]byte(acmeFile), &certificates)
	if nil != err {
		panic(err)
	}

	for _, i := range certificates.Certificates {
		decodedKey, err := base64.StdEncoding.DecodeString(i.Key)
		if err != nil {
			fmt.Println(err)
		}
		decodedCertificates, err := base64.StdEncoding.DecodeString(i.Certificate)
		if err != nil {
			fmt.Println(err)
		}
		splittedCertificates := strings.Split(string(decodedCertificates), "\n\n")

		certificatesProviders.Certificates[i.Domain.Main] = Certificate{
			certificate: splittedCertificates[0],
			key: string(decodedKey),
		}

		v, _ := tls.X509KeyPair([]byte(splittedCertificates[0]), decodedKey)
		tlsconfig.Certificates = append(tlsconfig.Certificates, v)
		*configChannel <- 1
	}
}

func initWatcher(certificatesProviders *CommonProvider, tlsconfig *tls.Config, configChannel *chan int) {
	watcher, err := fsnotify.NewWatcher()
	loadFromConfigFile(certificatesProviders, tlsconfig, configChannel)

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					loadFromConfigFile(certificatesProviders, tlsconfig, configChannel)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	err = watcher.Add(FILE_LOCATION)
	if err != nil {
		panic(err)
	}
	<-done
}

func TraefikInitProvider(certificates *CommonProvider, tlsconfig *tls.Config, configChannel *chan int)  {
	initWatcher(certificates, tlsconfig, configChannel)
}
