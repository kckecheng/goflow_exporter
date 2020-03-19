package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const (
	// ErrRC return code for error
	ErrRC = 1
	// Partition default consuming partition
	Partition = 0
)

// MQCfg message queue configuration, kafka is the only supported option now
var MQCfg mqCfg

type mqCfg struct {
	Brokers []string `json:"brokers"`
	Topic   string   `json:"topic"`
	Timeout uint8    `json:"timeout"`
}

func init() {
	getOptions()
}

// InitCfg get configuration options
func initCfg(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Fail to open configure file %s: %s", path, err.Error())
		os.Exit(ErrRC)
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Printf("Fail to read configure file %s: %s", path, err.Error())
		os.Exit(ErrRC)
	}

	err = json.Unmarshal(bytes, &MQCfg)
	if err != nil {
		fmt.Printf("Configure file %s is not a valid json file", path)
		os.Exit(ErrRC)
	}
}

func getHelp() {
	fmt.Printf("Usage: %s <-c|--config> <config file name>.json\n", os.Args[0])
	os.Exit(ErrRC)
}

func getOptions() {
	switch n := len(os.Args); n {
	case 2:
		if os.Args[1] != "-h" && os.Args[1] != "--help" {
			getHelp()
		}
	case 3:
		if os.Args[1] != "-c" && os.Args[1] != "--config" {
			getHelp()
		}
		initCfg(os.Args[2])
	default:
		getHelp()
	}

}

// ErrExit exit the execution
func ErrExit(msg string) {
	fmt.Println(msg)
	os.Exit(ErrRC)
}
