package providers

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/darkweak/souin/configuration"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"log"
	"strings"
)

// CommonProvider contains a Certificate map
type CommonProvider struct {
	Certificates map[string]Certificate
	fileLocation string
}

// Certificate contains key:certificate combo
type Certificate struct {
	certificate string
	key         string
}

// InitProviders function allow to init certificates and be able to exploit data as needed
func InitProviders(tlsconfig *tls.Config, configChannel *chan int, configuration *configuration.Configuration) {
	var providers []CommonProvider
	for _, provider := range configuration.GetSSLProviders() {
		providers = append(providers, CommonProvider{
			Certificates: make(map[string]Certificate),
			fileLocation: fmt.Sprintf("/ssl/%s.json", provider),
		})
	}
	for _, provider := range providers {
		provider.InitWatcher(tlsconfig, configChannel)
	}
}

// LoadFromConfigFile load SSL certs from one file by provider
func (c *CommonProvider) LoadFromConfigFile(tlsconfig *tls.Config, configChannel *chan int) {
	acmeFile, err := ioutil.ReadFile(c.fileLocation)
	if nil != err {
		return
	}

	certificates := &AcmeFile{}
	err = json.Unmarshal(acmeFile, &certificates)
	if nil != err {
		panic(err)
	}

	for _, i := range certificates.Certificates {
		decodedKey, er := base64.StdEncoding.DecodeString(i.Key)
		if er != nil {
			fmt.Println(er)
		}
		decodedCertificates, e := base64.StdEncoding.DecodeString(i.Certificate)
		if e != nil {
			fmt.Println(e)
		}
		splittedCertificates := strings.Split(string(decodedCertificates), "\n\n")

		c.Certificates[i.Domain.Main] = Certificate{
			certificate: splittedCertificates[0],
			key:         string(decodedKey),
		}

		v, _ := tls.X509KeyPair([]byte(splittedCertificates[0]), decodedKey)
		tlsconfig.Certificates = append(tlsconfig.Certificates, v)
		*configChannel <- 1
	}
}

// InitWatcher will start watcher on one ssl aggregator file
func (c *CommonProvider) InitWatcher(tlsconfig *tls.Config, configChannel *chan int) {
	//fmt.Printf("Start new watcher on %s", c.fileLocation)
	watcher, err := fsnotify.NewWatcher()
	c.LoadFromConfigFile(tlsconfig, configChannel)

	if err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					c.LoadFromConfigFile(tlsconfig, configChannel)
					_ = watcher.Add(c.fileLocation)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	err = watcher.Add(c.fileLocation)
	if err != nil {
		return
	}
	<-done
}
