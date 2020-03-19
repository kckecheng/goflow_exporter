package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kckecheng/goflow_exporter/common"
	"github.com/kckecheng/goflow_exporter/message"
	"github.com/kckecheng/goflow_exporter/metric"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := common.MQCfg

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGHUP)
	ticker := time.NewTicker(time.Duration(cfg.Timeout) * time.Second)
	go func() {
		for {
			select {
			case <-sc:
				common.ErrExit("Terminate the execution")
			case <-ticker.C:
				common.Logger.Debugf("Reset previous metric values after expiration(%d seconds)", cfg.Timeout)
				metric.Reset()
			}
		}
	}()

	fc, ec := message.GetRecords(cfg.Brokers, cfg.Topic)
	go func() {
		for {
			select {
			case err, ok := <-ec:
				if ok {
					if err != nil {
						common.Logger.Errorf("Hit an error while processing topic %s: %s", cfg.Topic, err.Error())
						common.Logger.Warn("Ignore the error")
					}
				} else {
					common.ErrExit("Kafka hits an internal error")
				}
			case record, ok := <-fc:
				if ok {
					go func(r *message.FlowRecord) {
						metric.UpdateMetric(record)
					}(record)
				} else {
					common.ErrExit("Kafka hits an internal error")
				}
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
