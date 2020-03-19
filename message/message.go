package message

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/kckecheng/goflow_exporter/common"

	kafka "github.com/Shopify/sarama"
	flow "github.com/cloudflare/goflow/pb"
	proto "github.com/golang/protobuf/proto"
)

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
	err := proto.Unmarshal(msg.Value, &fmsg)
	if err != nil {
		log.Printf("Fail to decode flow message, ignore this record")
		return nil, err
	}

	return &fmsg, nil
}

// extractRecord format sflow message into human readable structure
func extractRecord(fmsg *flow.FlowMessage) *FlowRecord {
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
	config.Net.KeepAlive = time.Duration(common.MQCfg.Timeout) * time.Second

	master, err := kafka.NewConsumer(brokers, config)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to make connection to brokers %v: %s", brokers, err.Error()))
	}

	topics, err := master.Topics()
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to list topics: %s", err.Error()))
	}

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

	consumer, err := master.ConsumePartition(topic, common.Partition, kafka.OffsetNewest)
	if err != nil {
		common.ErrExit(fmt.Sprintf("Fail to create consumer for topic %s with partition %d: %s", common.MQCfg.Topic, common.Partition, err.Error()))
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
						log.Printf("Hit an error while decoding sflow message: %s", err.Error())
						log.Println("Ignore this record silently")
					} else {
						record := extractRecord(fmsg)
						fc <- record
					}
				} else {
					log.Println("Kafka topic has been closed, exit")
					close(fc)
					return
				}
				// default:
				//   log.Printf("No record is gotten within %d seconds, sleep and retry", common.MQCfg.Timeout)
				//   time.Sleep(time.Duration(common.MQCfg.Timeout) * time.Second)
			}
		}
	}()

	return fc, ec
}
