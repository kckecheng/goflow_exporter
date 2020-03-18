package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	kafka "github.com/Shopify/sarama"
	flow "github.com/cloudflare/goflow/pb"
	proto "github.com/golang/protobuf/proto"
)

const (
	topic     = "flow-messages"
	partition = 0
	errRC     = 1
)

var cfg config

type config struct {
	Brokers []string `json:"brokers"`
	Topic   string   `json:"topic"`
	Timeout uint8    `json:"timeout"`
}

func initCfg(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Fail to open configure file %s: %s", path, err.Error())
		os.Exit(errRC)
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Printf("Fail to read configure file %s: %s", path, err.Error())
		os.Exit(errRC)
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		fmt.Printf("Configure file %s is not a valid json file", path)
		os.Exit(errRC)
	}
}

func getHelp() {
	fmt.Printf("Usage: %s <-c|--config> <config file name>.json\n", os.Args[0])
	os.Exit(errRC)
}

func getCfg() {
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

func decodeMsg(msg *kafka.ConsumerMessage) (*flow.FlowMessage, error) {
	var fmsg flow.FlowMessage
	err := proto.Unmarshal(msg.Value, &fmsg)
	if err != nil {
		log.Printf("Fail to decode flow message, ignore this record")
		return nil, err
	}

	return &fmsg, nil
}

func errExit(msg string) {
	fmt.Println(msg)
	os.Exit(errRC)
}

func main() {
	getCfg()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGHUP)

	config := kafka.NewConfig()
	config.Net.KeepAlive = time.Duration(cfg.Timeout) * time.Second

	master, err := kafka.NewConsumer(cfg.Brokers, config)
	if err != nil {
		errExit(fmt.Sprintf("Fail to make connection to brokers %v: %s", cfg.Brokers, err.Error()))
	}

	topics, err := master.Topics()
	if err != nil {
		errExit(fmt.Sprintf("Fail to list topics: %s", err.Error()))
	}

	existed := false
	for _, t := range topics {
		if t == cfg.Topic {
			existed = true
			break
		}
	}
	if !existed {
		errExit(fmt.Sprintf("Fail to find topic %s", cfg.Topic))
	}

	consumer, err := master.ConsumePartition(cfg.Topic, partition, kafka.OffsetNewest)
	if err != nil {
		errExit(fmt.Sprintf("Fail to create consumer for topic %s with partition %d: %s", cfg.Topic, partition, err.Error()))
	}

	mc := consumer.Messages()
	ec := consumer.Errors()
	for {
		select {
		case msg, ok := <-mc:
			if ok {
				fmsg, err := decodeMsg(msg)
				if err != nil {
					log.Printf("Hit an error while decoding sflow message: %s", err.Error())
				} else {
					log.Println(fmsg)
				}
			} else {
				log.Println("Kafka topic has been closed, exit")
				break
			}
		case err, ok := <-ec:
			if ok {
				if err != nil {
					log.Printf("Topic %s partition %d err: %s", topic, partition, err.Error())
					continue
				}
			} else {
				log.Println("Kafka topic has been closed, exit")
				break
			}
		case <-sc:
			log.Print("Terminate the application")
		}
	}
}
