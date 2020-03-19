package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	// ErrRC error out return code
	ErrRC = 1
	// Partition default consuming partition
	Partition = 0
)

// Logger global logger
var Logger *logrus.Logger

// MQCfg message queue configuration, kafka is the only supported system
var MQCfg mqCfg

type mqCfg struct {
	Brokers      []string `json:"brokers"`
	Topic        string   `json:"topic"`
	Timeout      uint8    `json:"timeout"`
	ExporterPort uint32   `json:"exporter_port"`
}

func init() {
	Logger = initLogger()
	MQCfg = getOptions()
}

func initLogger() *logrus.Logger {
	level := logrus.InfoLevel

	v, ok := os.LookupEnv("DEBUG")
	if ok {
		if v == "True" || v == "true" {
			level = logrus.DebugLevel
		}
	}

	formatter := logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	}
	logger := logrus.New()
	logger.SetFormatter(&formatter)
	logger.Out = os.Stdout
	logger.SetLevel(level)

	return logger
}

// InitCfg get configuration options
func parseCfg(path string, cfg *mqCfg) {
	f, err := os.Open(path)
	if err != nil {
		Logger.Fatalf("Fail to open configure file %s: %s", path, err.Error())
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		Logger.Fatalf("Fail to read configure file %s: %s", path, err.Error())
	}

	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		Logger.Fatalf("Configure file %s is not a valid json file", path)
	}
}

func getHelp() {
	ErrExit(fmt.Sprintf("Usage: %s <-c|--config> <config file name>.json", os.Args[0]))
}

func getOptions() mqCfg {
	var cfg mqCfg

	switch n := len(os.Args); n {
	case 2:
		if os.Args[1] != "-h" && os.Args[1] != "--help" {
			getHelp()
		}
	case 3:
		if os.Args[1] != "-c" && os.Args[1] != "--config" {
			getHelp()
		}
		parseCfg(os.Args[2], &cfg)
	default:
		getHelp()
	}

	return cfg
}

// ErrExit exit the execution
func ErrExit(msg string) {
	fmt.Println(msg)
	os.Exit(ErrRC)
}
