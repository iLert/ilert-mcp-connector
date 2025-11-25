package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type KafkaHandler struct {
	config KafkaConfig
	client *kafka.Client
}

func NewKafkaHandler(config KafkaConfig) *KafkaHandler {
	client := &kafka.Client{
		Addr:    kafka.TCP(config.Brokers...),
		Timeout: 10 * time.Second,
	}

	return &KafkaHandler{
		config: config,
		client: client,
	}
}

func checkKafkaConnection(config KafkaConfig) error {
	client := &kafka.Client{
		Addr:    kafka.TCP(config.Brokers...),
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := kafka.MetadataRequest{
		Topics: []string{},
	}
	_, err := client.Metadata(ctx, &req)
	return err
}

func (h *KafkaHandler) ListTopics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := kafka.MetadataRequest{
		Topics: []string{},
	}
	resp, err := h.client.Metadata(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	topics := make([]TopicInfo, 0, len(resp.Topics))
	for _, topic := range resp.Topics {
		topicInfo := TopicInfo{
			Name:       topic.Name,
			Partitions: len(topic.Partitions),
		}
		if topic.Error != nil {
			errStr := topic.Error.Error()
			topicInfo.Error = &errStr
		}
		topics = append(topics, topicInfo)
	}

	c.JSON(http.StatusOK, TopicsResponse{Topics: topics})
}

func (h *KafkaHandler) DescribeTopic(c *gin.Context) {
	topicName := c.Param("topic")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := kafka.MetadataRequest{
		Topics: []string{topicName},
	}
	resp, err := h.client.Metadata(ctx, &req)
	if err != nil {
		log.Error().Err(err).Str("topic", topicName).Msg("Failed to describe Kafka topic")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	if len(resp.Topics) == 0 {
		log.Warn().Str("topic", topicName).Msg("Topic not found")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "topic not found"})
		return
	}

	topic := resp.Topics[0]
	partitions := make([]PartitionInfo, 0, len(topic.Partitions))
	for _, partition := range topic.Partitions {
		partitions = append(partitions, PartitionInfo{
			ID:       partition.ID,
			Leader:   partition.Leader.ID,
			Replicas: getReplicaIDs(partition.Replicas),
			ISR:      getReplicaIDs(partition.Isr),
		})
	}

	response := TopicDetailResponse{
		Name:       topic.Name,
		Partitions: partitions,
	}
	if topic.Error != nil {
		errStr := topic.Error.Error()
		response.Error = &errStr
	}
	c.JSON(http.StatusOK, response)
}

func (h *KafkaHandler) ListConsumers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := kafka.ListGroupsRequest{}
	resp, err := h.client.ListGroups(ctx, &req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list Kafka consumer groups")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	groups := make([]ConsumerGroupInfo, 0, len(resp.Groups))
	for _, group := range resp.Groups {
		groups = append(groups, ConsumerGroupInfo{
			GroupID: group.GroupID,
		})
	}

	c.JSON(http.StatusOK, ConsumersResponse{Consumers: groups})
}

func (h *KafkaHandler) DescribeConsumer(c *gin.Context) {
	groupID := c.Param("group")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := kafka.DescribeGroupsRequest{
		GroupIDs: []string{groupID},
	}
	resp, err := h.client.DescribeGroups(ctx, &req)
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to describe Kafka consumer group")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	if len(resp.Groups) == 0 {
		log.Warn().Str("group", groupID).Msg("Consumer group not found")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "consumer group not found"})
		return
	}

	group := resp.Groups[0]
	members := make([]MemberInfo, 0, len(group.Members))
	for _, member := range group.Members {
		memberInfo := MemberInfo{
			MemberID:   member.MemberID,
			ClientID:   member.ClientID,
			ClientHost: member.ClientHost,
		}
		if len(member.MemberMetadata.Topics) > 0 || len(member.MemberMetadata.UserData) > 0 || len(member.MemberMetadata.OwnedPartitions) > 0 {
			ownedPartitions := make([]OwnedPartitionInfo, 0, len(member.MemberMetadata.OwnedPartitions))
			for _, p := range member.MemberMetadata.OwnedPartitions {
				ownedPartitions = append(ownedPartitions, OwnedPartitionInfo{
					Topic:      p.Topic,
					Partitions: p.Partitions,
				})
			}
			memberInfo.MemberMetadata = &MemberMetadataInfo{
				Version:         member.MemberMetadata.Version,
				Topics:          member.MemberMetadata.Topics,
				UserData:        member.MemberMetadata.UserData,
				OwnedPartitions: ownedPartitions,
			}
		}
		members = append(members, memberInfo)
	}

	response := ConsumerGroupDetailResponse{
		GroupID: group.GroupID,
		Members: members,
	}
	if group.Error != nil {
		errStr := group.Error.Error()
		response.Error = &errStr
	}
	c.JSON(http.StatusOK, response)
}

func (h *KafkaHandler) GetConsumerLag(c *gin.Context) {
	groupID := c.Param("group")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, get metadata to find all topics
	metadataReq := kafka.MetadataRequest{
		Topics: []string{},
	}
	metadataResp, err := h.client.Metadata(ctx, &metadataReq)
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to get metadata for consumer lag")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// Get consumer group offsets
	offsetReq := kafka.OffsetFetchRequest{
		GroupID: groupID,
		Topics:  make(map[string][]int),
	}

	// Build offset fetch request for all topics
	for _, topic := range metadataResp.Topics {
		partitions := make([]int, 0, len(topic.Partitions))
		for _, partition := range topic.Partitions {
			partitions = append(partitions, partition.ID)
		}
		offsetReq.Topics[topic.Name] = partitions
	}

	offsetResp, err := h.client.OffsetFetch(ctx, &offsetReq)
	if err != nil {
		log.Error().Err(err).Str("group", groupID).Msg("Failed to fetch offsets for consumer lag")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	// Get high water marks for each partition
	lagData := make([]ConsumerLagInfo, 0)
	for topicName, partitions := range offsetResp.Topics {
		for partitionID, offset := range partitions {
			// Get high water mark
			highWaterReq := kafka.ListOffsetsRequest{
				Topics: map[string][]kafka.OffsetRequest{
					topicName: {
						{Partition: partitionID, Timestamp: -1}, // -1 means latest
					},
				},
			}
			highWaterResp, err := h.client.ListOffsets(ctx, &highWaterReq)
			if err != nil {
				continue
			}

			var highWaterMark int64
			if topicOffsets, ok := highWaterResp.Topics[topicName]; ok {
				if len(topicOffsets) > 0 {
					highWaterMark = topicOffsets[0].LastOffset
				}
			}

			lag := highWaterMark - offset.CommittedOffset
			if lag < 0 {
				lag = 0
			}

			lagData = append(lagData, ConsumerLagInfo{
				Topic:         topicName,
				Partition:     partitionID,
				CurrentOffset: offset.CommittedOffset,
				HighWaterMark: highWaterMark,
				Lag:           lag,
			})
		}
	}

	c.JSON(http.StatusOK, ConsumerLagResponse{
		GroupID: groupID,
		Lag:     lagData,
	})
}

func getReplicaIDs(replicas []kafka.Broker) []int {
	ids := make([]int, 0, len(replicas))
	for _, replica := range replicas {
		ids = append(ids, replica.ID)
	}
	return ids
}
