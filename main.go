package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kckecheng/goflow_exporter/common"
	"github.com/kckecheng/goflow_exporter/message"
)

func main() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGHUP)

	cfg := common.MQCfg

	fc, ec := message.GetRecords(cfg.Brokers, cfg.Topic)
	for {
		select {
		case record, ok := <-fc:
			if ok {
				log.Printf("%+v", record)
			} else {
				common.ErrExit("Kafka hits an error, exit")
			}
		case err, ok := <-ec:
			if ok {
				if err != nil {
					log.Printf("Topic %s partition %d err: %s", cfg.Topic, common.Partition, err.Error())
					break
				}
			} else {
				common.ErrExit("Kafka hits an error, exit")
			}
		case <-sc:
			common.ErrExit("Terminate the execution")
		}
	}
}
