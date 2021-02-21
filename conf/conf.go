package conf

import (
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
	TargetAddr string `json:"TistenAddr"`
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
