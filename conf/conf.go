package conf

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"os"
)

// ServerConf contains the server configuration addresses
type ServerConf struct {
	ListenAddr string `json:"listenAddr"`
	EventAddr  string `json:"eventAddr"`
}

// ReadServerConf reads the config file
func ReadServerConf() (ServerConf, error) {
	var conf ServerConf
	f, err := os.Open("./server.json")
	if err != nil {
		return conf, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}
	err = json.Unmarshal(data, &conf)
	return conf, err
}

// ClientConf contains the server configuration addresses
type ClientConf struct {
	EventAddr  string `json:"eventAddr"`
	TargetAddr string `json:"targetAddr"`
}

// GetTLSConfig returns the TLS configuration with direction from -> to
func GetTLSConfig(from string, to string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("./cert/"+from+".crt", "./cert/"+from+".key")
	if err != nil {
		return nil, err
	}
	myPool := x509.NewCertPool()
	{
		b, err := ioutil.ReadFile("./cert/" + to + ".crt")
		if err != nil {
			return nil, err
		}
		myPool.AppendCertsFromPEM(b)
	}
	tlsConf := &tls.Config{
		RootCAs:            myPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          myPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	return tlsConf, nil
}

// ReadClientConf reads the config file
func ReadClientConf() (ClientConf, error) {
	var conf ClientConf
	f, err := os.Open("./client.json")
	if err != nil {
		return conf, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}
	err = json.Unmarshal(data, &conf)
	return conf, err
}
