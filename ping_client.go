package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// PingClient is a client that can be started
type PingClient interface {
	Start()
}

type pingClient struct {
	addresses []string
	client    *http.Client
	interval  time.Duration
	logger    *zap.SugaredLogger
}

func (pc *pingClient) Start() {
	c := time.Tick(time.Duration(pc.interval) * time.Second)
	for range c {
		for _, address := range pc.addresses {
			pc.pingAddress(address)
		}
	}
}

func (pc *pingClient) pingAddress(address string) {
	response, err := pc.client.Get(address + "/ping")
	if err != nil {
		pc.logger.Infof("Couldn't ping %s: %s", address, err)
	} else {
		defer response.Body.Close()
		if response.StatusCode != 200 {
			pc.logger.Errorf("Got non-ok response code: '%v'", response.StatusCode)
		} else {
			var pingAnswer string
			err := json.NewDecoder(response.Body).Decode(&pingAnswer)
			if err != nil {
				pc.logger.Errorf("error decoding json response: '%v'", err)
			}
			pc.logger.Infof("Pinged %s, got response: %s, \"%s\"", address, response.Status, pingAnswer)
		}
	}
}

// NewPingClient creates a new client for pinging PingServers
func NewPingClient(serverConfig *HTTPConfig, addresses []string, interval time.Duration) PingClient {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	sugar := logger.Sugar()

	client := http.DefaultClient
	if serverConfig.caCertFile != "" {
		caCert, err := ioutil.ReadFile(serverConfig.caCertFile)
		if err != nil {
			sugar.Fatal(err)
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			sugar.Fatal(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		client = &http.Client{Transport: transport}
	}

	return &pingClient{addresses, client, interval, sugar}
}
