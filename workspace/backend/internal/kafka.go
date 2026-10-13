package internal

// import (
// 	"context"
// 	"fmt"
// 	"main/utils"
// 	"time"

// 	"github.com/confluentinc/confluent-kafka-go/kafka"
// )

// type KafkaInstance struct {
// 	kafkaProducer *kafka.Producer
// 	kafkaConsumer *kafka.Consumer
// }

// func NewKafkaInstance(topics []utils.KafkaTopic) *KafkaInstance {
// 	return &KafkaInstance{
// 		kafkaProducer: newKafkaProducer(topics),
// 		kafkaConsumer: newKafkaConsumer(),
// 	}
// }

// func newKafkaProducer(topics []utils.KafkaTopic) *kafka.Producer {
// 	config := utils.AppConfig.Kafka
// 	p, err := kafka.NewProducer(&kafka.ConfigMap{
// 		"bootstrap.servers": config.Producer.BootstrapServer,
// 		"acks":              config.Producer.Acks,
// 		"security.protocol": config.Producer.SecurityProtocol,
// 	})
// 	if err != nil {
// 		utils.ShowErrorLogs(err)
// 		return nil
// 	}

// 	// Create topic if needed
// 	for _, topic := range topics {
// 		err = createTopic(p, topic.Name, topic.Partition)
// 		if err != nil {
// 			utils.ShowErrorLogs(err)
// 			return nil
// 		}
// 	}

// 	// Delivery report handler for produced messages
// 	go func() {
// 		for e := range p.Events() {
// 			switch ev := e.(type) {
// 			case *kafka.Message:
// 				if ev.TopicPartition.Error != nil {
// 					utils.ShowInfoLogs(fmt.Sprintf("Delivery failed: %v\n", ev.TopicPartition))
// 				} else {
// 					utils.ShowInfoLogs(fmt.Sprintf("Delivered message to %v\n", ev.TopicPartition))
// 				}
// 			}
// 		}
// 	}()

// 	utils.ShowInfoLogs(fmt.Sprintf("Connected with Kafka Producer %s\n", config.Producer.BootstrapServer))

// 	return p
// }

// func newKafkaConsumer() *kafka.Consumer {
// 	config := utils.AppConfig.Kafka
// 	c, err := kafka.NewConsumer(&kafka.ConfigMap{
// 		"bootstrap.servers": config.Consumer.BootstrapServer,
// 		"group.id":          config.Consumer.GroupId,
// 		"security.protocol": config.Consumer.SecurityProtocol,
// 	})
// 	if err != nil {
// 		utils.ShowErrorLogs(err)
// 	} else {
// 		utils.ShowInfoLogs(fmt.Sprintf("Connected with Kafka Consumer %s\n", config.Consumer.BootstrapServer))
// 	}

// 	return c
// }

// func createTopic(p *kafka.Producer, topic string, numberOfPartition int) error {
// 	a, err := kafka.NewAdminClientFromProducer(p)
// 	if err != nil {
// 		fmt.Printf("Failed to create new admin client from producer: %s", err)
// 		return err
// 	}

// 	// Contexts are used to abort or limit the amount of time
// 	// the Admin call blocks waiting for a result.
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	// Create topics on cluster.
// 	// Set Admin options to wait up to 60s for the operation to finish on the remote cluster
// 	maxDur, err := time.ParseDuration("60s")
// 	if err != nil {
// 		fmt.Printf("ParseDuration(60s): %s", err)
// 		return err
// 	}
// 	results, err := a.CreateTopics(
// 		ctx,
// 		// Multiple topics can be created simultaneously
// 		// by providing more TopicSpecification structs here.
// 		[]kafka.TopicSpecification{{
// 			Topic:         topic,
// 			NumPartitions: numberOfPartition,
// 		}},
// 		// Admin options
// 		kafka.SetAdminOperationTimeout(maxDur))
// 	if err != nil {
// 		fmt.Printf("Admin Client request error: %v\n", err)
// 		return err
// 	}
// 	for _, result := range results {
// 		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
// 			fmt.Printf("Failed to create topic: %v", result.Error)
// 			return err
// 		}
// 		fmt.Printf("Created topic %v success", result)
// 	}
// 	return nil
// }

// func (k *KafkaInstance) PushMessage(topic string, partition int, data []byte) error {
// 	err := k.kafkaProducer.Produce(&kafka.Message{
// 		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: int32(partition)},
// 		Value:          data,
// 	}, nil)
// 	return err
// }

// func (k *KafkaInstance) Subscribe(topic string, processFunc func([]byte)) error {
// 	err := k.kafkaConsumer.Subscribe(topic, nil)
// 	if err != nil {
// 		return err
// 	}

// 	go func() {
// 		for {
// 			msg, err := k.kafkaConsumer.ReadMessage(-1)
// 			if err != nil {
// 				utils.ShowErrorLogs(fmt.Errorf("consumer error: %v (%v)", err, msg))
// 			} else {
// 				processFunc(msg.Value)
// 			}
// 		}
// 	}()
// 	return nil
// }
