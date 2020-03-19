package main

import (
	"fmt"
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
						metric.UpdateMetric(record)
					}(record)
				} else {
					common.ErrExit("Kafka hits an internal error")
				}
			case err, ok := <-ec:
				if ok {
					if err != nil {
						common.Logger.Errorf("Get an error on topic %s with partition %d: %s", cfg.Topic, common.Partition, err.Error())
						common.Logger.Warn("Ignore the error")
					}
				} else {
					common.ErrExit("Kafka hits an internal error")
				}
			case <-sc:
				common.ErrExit("Terminate the execution")
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())

	common.Logger.Infof("Access http://localhost:%d/metrics for metrics", cfg.ExporterPort)
	common.Logger.Info("To view more detailed output: export DEBUG=true")

	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.ExporterPort), nil)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to start HTTP server: %s", err.Error()))
	}
}
