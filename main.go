package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kckecheng/goflow_exporter/common"
	"github.com/kckecheng/goflow_exporter/message"
	"github.com/kckecheng/goflow_exporter/metric"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGHUP)

	cfg := common.MQCfg

	fc, ec := message.GetRecords(cfg.Brokers, cfg.Topic)
	go func() {
		for {
			select {
			case record, ok := <-fc:
				if ok {
					go func(r *message.FlowRecord) {
						log.Printf("Get metric: %+v", record)
						metric.UpdateMetric(record)
					}(record)
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
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Access http://localhost:%d/metrics for metrics", cfg.ExporterPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.ExporterPort), nil)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to start HTTP server: %s", err.Error()))
	}
}
