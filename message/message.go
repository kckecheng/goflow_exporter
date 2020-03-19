package message

import (
	"fmt"
	"net"
	"time"

	"github.com/kckecheng/goflow_exporter/common"

	kafka "github.com/Shopify/sarama"
	flow "github.com/cloudflare/goflow/pb"
	proto "github.com/golang/protobuf/proto"
)

const partition = 0

// FlowRecord human readable representation for sflow message
type FlowRecord struct {
	Type          string
	TimeRecvd     time.Time
	SamplingRate  uint64
	SequenceNum   uint32
	TimeFlow      time.Time
	SrcIP         string
	DstIP         string
	Bytes         uint64
	Packets       uint64
	RouterAddr    string
	SrcIf         uint32
	DstIf         uint32
	Proto         uint32
	SrcPort       uint32
	DstPort       uint32
	IPTTL         uint32
	TCPFlags      uint32
	SrcMac        uint64
	DstMac        uint64
	Etype         uint32
	IPv6FlowLabel uint32
	IngressVrfID  uint32
}

// decodeMsg decode kafka message into sflow message
func decodeMsg(msg *kafka.ConsumerMessage) (*flow.FlowMessage, error) {
	var fmsg flow.FlowMessage
	common.Logger.Debugf("Decode Kafka message into sflow message: %+v", msg)
	err := proto.Unmarshal(msg.Value, &fmsg)
	if err != nil {
		common.Logger.Error("Fail to decode sflow message")
		return nil, err
	}

	return &fmsg, nil
}

// extractRecord format sflow message into human readable structure
func extractRecord(fmsg *flow.FlowMessage) *FlowRecord {
	common.Logger.Debugf("Extract sflow message into sflow record: %+v", fmsg)
	// Refer to https://sflow.org/sflow_version_5.txt for detailed field meaning
	record := FlowRecord{
		Type:         fmsg.Type.String(),
		TimeRecvd:    time.Unix(int64(fmsg.GetTimeRecvd()), 0),
		SamplingRate: fmsg.GetSamplingRate(),
		SequenceNum:  fmsg.GetSequenceNum(),
		TimeFlow:     time.Unix(int64(fmsg.GetTimeFlow()), 0),
		SrcIP:        net.IP(fmsg.GetSrcIP()).String(),
		DstIP:        net.IP(fmsg.GetDstIP()).String(),
		Bytes:        fmsg.GetBytes(),
		Packets:      fmsg.GetPackets(),
		SrcIf:        fmsg.GetSrcIf(),
		RouterAddr:   string(fmsg.GetRouterAddr()),
		DstIf:        fmsg.GetDstIf(),
		Proto:        fmsg.GetProto(), // Layer 4 protocol
		SrcPort:      fmsg.GetSrcPort(),
		DstPort:      fmsg.GetDstPort(),
		Etype:        fmsg.GetEtype(), // Layer 3 protocol
		// IPTTL:         fmsg.GetIPTTL(),
		// TCPFlags:      fmsg.GetTCPFlags(),
		// SrcMac:        fmsg.GetSrcMac(),
		// DstMac:        fmsg.GetDstMac(),
		// IPv6FlowLabel: fmsg.GetIPv6FlowLabel(),
		// IngressVrfID:  fmsg.GetIngressVrfId(),
	}
	return &record
}

// getConsumer connect to Kafka brokers and return a consumer for a specified topic
func getConsumer(brokers []string, topic string) kafka.PartitionConsumer {
	config := kafka.NewConfig()
	config.Net.KeepAlive = 10 * time.Second

	common.Logger.Debugf("Connect to Kafka brokers %v", brokers)
	master, err := kafka.NewConsumer(brokers, config)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to make connection to brokers %v: %s", brokers, err.Error()))
	}

	topics, err := master.Topics()
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to list topics: %s", err.Error()))
	}
	common.Logger.Debugf("Find Kafka topics: %v", topics)

	existed := false
	for _, t := range topics {
		if t == topic {
			existed = true
			break
		}
	}
	if !existed {
		common.ErrExit(fmt.Sprintf("Fail to find topic %s", topic))
	}

	common.Logger.Debugf("Create consumer for topic %s with parition %d", topic, partition)
	consumer, err := master.ConsumePartition(topic, partition, kafka.OffsetNewest)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to create consumer for topic %s with partition %d: %s", common.MQCfg.Topic, partition, err.Error()))
	}

	return consumer
}

// GetRecords get sflow records and errors
func GetRecords(brokers []string, topic string) (<-chan *FlowRecord, <-chan *kafka.ConsumerError) {
	consumer := getConsumer(brokers, topic)

	mc := consumer.Messages()
	fc := make(chan *FlowRecord)
	ec := consumer.Errors()

	go func() {
		for {
			select {
			case msg, ok := <-mc:
				if ok {
					fmsg, err := decodeMsg(msg)
					if err != nil {
						common.Logger.Error(err)
						common.Logger.Warn("Ignore this sflow message silently")
					} else {
						record := extractRecord(fmsg)
						common.Logger.Debugf("Get sflow record: %+v", record)
						fc <- record
					}
				} else {
					common.Logger.Error("Kafka hit an internal error")
					close(fc)
					return
				}
				// default:
				//   common.Logger.Infof("No record is gotten within %d seconds, sleep and retry", common.MQCfg.Timeout)
				//   time.Sleep(time.Duration(common.MQCfg.Timeout) * time.Second)
			}
		}
	}()

	return fc, ec
}
