package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type KafkaHandler struct {
	config      KafkaConfig
	adminClient *kafka.AdminClient
}

func buildKafkaConfigMap(config KafkaConfig, timeoutMs int) kafka.ConfigMap {
	configMap := kafka.ConfigMap{
		"bootstrap.servers":           strings.Join(config.Brokers, ","),
		"client.id":                   config.ClientID,
		"metadata.request.timeout.ms": timeoutMs,
	}

	hasSSL := config.SSLCertLocation != "" && config.SSLKeyLocation != ""
	if hasSSL {
		configMap["ssl.certificate.location"] = config.SSLCertLocation
		configMap["ssl.key.location"] = config.SSLKeyLocation
		if config.SSLCALocation != "" {
			configMap["ssl.ca.location"] = config.SSLCALocation
		}
		if config.SSLKeyPassword != "" {
			configMap["ssl.key.password"] = config.SSLKeyPassword
		}
		if config.SSLInsecureSkipTLS {
			configMap["enable.ssl.certificate.verification"] = false
		}
	}

	switch config.AuthType {
	case "ssl":
		if !hasSSL {
			log.Warn().Msg("KAFKA_AUTH_TYPE=ssl requires SSL certificates. Set KAFKA_SSL_CERT_LOCATION and KAFKA_SSL_KEY_LOCATION")
		} else {
			configMap["security.protocol"] = "ssl"
		}
	case "sasl_ssl":
		if !hasSSL {
			log.Warn().Msg("KAFKA_AUTH_TYPE=sasl_ssl requires SSL certificates. Set KAFKA_SSL_CERT_LOCATION and KAFKA_SSL_KEY_LOCATION")
		} else {
			configMap["security.protocol"] = "sasl_ssl"
		}
		fallthrough
	case "plain":
		if config.AuthType == "plain" {
			configMap["security.protocol"] = "SASL_PLAINTEXT"
		}
		configMap["sasl.mechanism"] = "PLAIN"
		configMap["sasl.username"] = config.Username
		configMap["sasl.password"] = config.Password
	case "scram-sha-256":
		configMap["security.protocol"] = "SASL_PLAINTEXT"
		configMap["sasl.mechanism"] = "SCRAM-SHA-256"
		configMap["sasl.username"] = config.Username
		configMap["sasl.password"] = config.Password
	case "scram-sha-512":
		configMap["security.protocol"] = "SASL_PLAINTEXT"
		configMap["sasl.mechanism"] = "SCRAM-SHA-512"
		configMap["sasl.username"] = config.Username
		configMap["sasl.password"] = config.Password
	case "none", "":
	default:
		log.Warn().Str("authType", config.AuthType).Msg("Unknown Kafka auth type, using no authentication")
	}

	return configMap
}

func NewKafkaHandler(config KafkaConfig) *KafkaHandler {
	configMap := buildKafkaConfigMap(config, 10000)

	adminClient, err := kafka.NewAdminClient(&configMap)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Kafka admin client")
		return &KafkaHandler{
			config:      config,
			adminClient: nil,
		}
	}

	return &KafkaHandler{
		config:      config,
		adminClient: adminClient,
	}
}

func checkKafkaConnection(config KafkaConfig) error {
	configMap := buildKafkaConfigMap(config, 5000)

	adminClient, err := kafka.NewAdminClient(&configMap)
	if err != nil {
		return err
	}
	defer adminClient.Close()

	metadata, err := adminClient.GetMetadata(nil, true, int(5*time.Second.Milliseconds()))
	if err != nil {
		return err
	}

	if len(metadata.Brokers) == 0 {
		return fmt.Errorf("no brokers found")
	}

	return nil
}

func (h *KafkaHandler) ListTopics(c *gin.Context) {
	if h.adminClient == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Kafka admin client not initialized"})
		return
	}

	metadata, err := h.adminClient.GetMetadata(nil, true, int(10*time.Second.Milliseconds()))
	if err != nil {
		log.Error().Err(err).Msg("Failed to list Kafka topics")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	topics := make([]TopicInfo, 0, len(metadata.Topics))
	for topicName, topicMetadata := range metadata.Topics {
		topicInfo := TopicInfo{
			Name:       topicName,
			Partitions: len(topicMetadata.Partitions),
		}
		if topicMetadata.Error.Code() != kafka.ErrNoError {
			errStr := topicMetadata.Error.String()
			topicInfo.Error = &errStr
		}
		topics = append(topics, topicInfo)
	}

	c.JSON(http.StatusOK, TopicsResponse{Topics: topics})
}

func (h *KafkaHandler) DescribeTopic(c *gin.Context) {
	if h.adminClient == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Kafka admin client not initialized"})
		return
	}

	topicName := c.Param("topic")

	metadata, err := h.adminClient.GetMetadata(&topicName, true, int(10*time.Second.Milliseconds()))
	if err != nil {
		log.Error().Err(err).Str("topic", topicName).Msg("Failed to describe Kafka topic")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	topicMetadata, exists := metadata.Topics[topicName]
	if !exists {
		log.Warn().Str("topic", topicName).Msg("Topic not found")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "topic not found"})
		return
	}

	partitions := make([]PartitionInfo, 0, len(topicMetadata.Partitions))
	for partitionID, partitionMetadata := range topicMetadata.Partitions {
		replicas := make([]int, 0, len(partitionMetadata.Replicas))
		for _, replicaID := range partitionMetadata.Replicas {
			replicas = append(replicas, int(replicaID))
		}

		isr := make([]int, 0, len(partitionMetadata.Isrs))
		for _, isrID := range partitionMetadata.Isrs {
			isr = append(isr, int(isrID))
		}

		partitions = append(partitions, PartitionInfo{
			ID:       partitionID,
			Leader:   int(partitionMetadata.Leader),
			Replicas: replicas,
			ISR:      isr,
		})
	}

	response := TopicDetailResponse{
		Name:       topicName,
		Partitions: partitions,
	}
	if topicMetadata.Error.Code() != kafka.ErrNoError {
		errStr := topicMetadata.Error.String()
		response.Error = &errStr
	}

	c.JSON(http.StatusOK, response)
}

func (h *KafkaHandler) ListConsumers(c *gin.Context) {
	if h.adminClient == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Kafka admin client not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := h.adminClient.ListConsumerGroups(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list Kafka consumer groups")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	consumerGroups := make([]ConsumerGroupInfo, 0, len(result.Valid))
	for _, group := range result.Valid {
		consumerGroups = append(consumerGroups, ConsumerGroupInfo{
			GroupID: group.GroupID,
		})
	}

	c.JSON(http.StatusOK, ConsumersResponse{Consumers: consumerGroups})
}

func (h *KafkaHandler) DescribeConsumer(c *gin.Context) {
	if h.adminClient == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Kafka admin client not initialized"})
		return
	}

	groupID := c.Param("group")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := h.adminClient.DescribeConsumerGroups(ctx, []string{groupID})
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to describe Kafka consumer group")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	if len(result.ConsumerGroupDescriptions) == 0 {
		log.Warn().Str("group", groupID).Msg("Consumer group not found")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "consumer group not found"})
		return
	}

	groupDesc := result.ConsumerGroupDescriptions[0]
	if groupDesc.Error.Code() != kafka.ErrNoError {
		errStr := groupDesc.Error.String()
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: errStr})
		return
	}

	members := make([]MemberInfo, 0, len(groupDesc.Members))
	for _, member := range groupDesc.Members {
		memberInfo := MemberInfo{
			MemberID:   member.ConsumerID,
			ClientID:   member.ClientID,
			ClientHost: member.Host,
		}

		ownedPartitions := make([]OwnedPartitionInfo, 0)
		memberInfo.MemberMetadata = &MemberMetadataInfo{
			Version:         0,
			Topics:          []string{},
			UserData:        []byte{},
			OwnedPartitions: ownedPartitions,
		}

		members = append(members, memberInfo)
	}

	response := ConsumerGroupDetailResponse{
		GroupID: groupID,
		Members: members,
	}

	c.JSON(http.StatusOK, response)
}

func (h *KafkaHandler) GetConsumerLag(c *gin.Context) {
	if h.adminClient == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Kafka admin client not initialized"})
		return
	}

	groupID := c.Param("group")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metadata, err := h.adminClient.GetMetadata(nil, true, int(10*time.Second.Milliseconds()))
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to get metadata for consumer lag")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	allPartitions := make([]kafka.TopicPartition, 0)
	for topicName, topicMetadata := range metadata.Topics {
		for partitionID := range topicMetadata.Partitions {
			topicNameCopy := topicName
			allPartitions = append(allPartitions, kafka.TopicPartition{
				Topic:     &topicNameCopy,
				Partition: int32(partitionID),
			})
		}
	}

	if len(allPartitions) == 0 {
		c.JSON(http.StatusOK, ConsumerLagResponse{
			GroupID: groupID,
			Lag:     []ConsumerLagInfo{},
		})
		return
	}

	offsetSpecs := make(map[kafka.TopicPartition]kafka.OffsetSpec)
	for _, tp := range allPartitions {
		offsetSpecs[tp] = kafka.NewOffsetSpecForTimestamp(-1)
	}

	highWaterMarks, err := h.adminClient.ListOffsets(ctx, offsetSpecs)
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to get high water marks for consumer lag")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	lagData := make([]ConsumerLagInfo, 0)

	configMap := buildKafkaConfigMap(h.config, 10000)
	configMap["group.id"] = groupID
	configMap["enable.auto.commit"] = false

	consumer, err := kafka.NewConsumer(&configMap)
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to create Kafka consumer for lag calculation")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	defer consumer.Close()

	committed, err := consumer.Committed(allPartitions, int(10*time.Second.Milliseconds()))
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to get committed offsets")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	committedMap := make(map[string]map[int32]kafka.Offset)
	for _, tp := range committed {
		topicName := ""
		if tp.Topic != nil {
			topicName = *tp.Topic
		}
		if committedMap[topicName] == nil {
			committedMap[topicName] = make(map[int32]kafka.Offset)
		}
		committedOffset := tp.Offset
		if committedOffset < 0 {
			committedOffset = 0
		}
		committedMap[topicName][tp.Partition] = committedOffset
	}

	for _, tp := range allPartitions {
		topicName := ""
		if tp.Topic != nil {
			topicName = *tp.Topic
		}

		committedOffset := kafka.Offset(0)
		if topicMap, ok := committedMap[topicName]; ok {
			if offset, ok := topicMap[tp.Partition]; ok {
				committedOffset = offset
			}
		}

		highWaterMarkInfo, ok := highWaterMarks.ResultInfos[tp]
		if !ok || highWaterMarkInfo.Error.Code() != kafka.ErrNoError {
			continue
		}

		highWaterMark := highWaterMarkInfo.Offset
		lag := highWaterMark - committedOffset
		if lag < 0 {
			lag = 0
		}

		lagData = append(lagData, ConsumerLagInfo{
			Topic:         topicName,
			Partition:     int(tp.Partition),
			CurrentOffset: int64(committedOffset),
			HighWaterMark: int64(highWaterMark),
			Lag:           int64(lag),
		})
	}

	c.JSON(http.StatusOK, ConsumerLagResponse{
		GroupID: groupID,
		Lag:     lagData,
	})
}
