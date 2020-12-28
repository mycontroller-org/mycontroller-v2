package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jaegertracing/jaeger/pkg/queue"
	"github.com/mycontroller-org/backend/v2/pkg/mcbus"
	"github.com/mycontroller-org/backend/v2/pkg/model"
	gwml "github.com/mycontroller-org/backend/v2/pkg/model/gateway"
	msgml "github.com/mycontroller-org/backend/v2/pkg/model/message"
	"github.com/mycontroller-org/backend/v2/pkg/utils"
	busml "github.com/mycontroller-org/backend/v2/plugin/bus"
	"github.com/mycontroller-org/backend/v2/plugin/gw_provider/mysensors"
	"github.com/mycontroller-org/backend/v2/plugin/gw_provider/tasmota"
	"go.uber.org/zap"
)

const (
	messageQueueSize          = 200
	rawMessageQueueSize       = 200
	sleepingQueueLimit        = 100
	numberOfWorkers           = 1
	rawMessageNumberOfWorkers = 1
)

// Service component of the provider
type Service struct {
	GatewayConfig                     *gwml.Config
	provider                          Provider
	messageQueue                      *queue.BoundedQueue
	rawMessageQueue                   *queue.BoundedQueue
	topicListenFromCore               string
	topicPostToCore                   string
	topicListenFromCoreSubscriptionID int64
	sleepingMessageQueue              map[string][]*msgml.Message
	mutex                             sync.RWMutex
	ctx                               context.Context
}

// SubscriptionID of topics
type SubscriptionID struct {
}

// GetService returns service instance
func GetService(gatewayCfg *gwml.Config) (*Service, error) {
	var provider Provider
	switch gatewayCfg.Provider.GetString(model.NameType) {
	case TypeMySensors:
		provider = mysensors.Init(gatewayCfg)

	case TypeTasmota:
		provider = tasmota.Init(gatewayCfg)

	default:
		return nil, fmt.Errorf("Unknown provider:%s", gatewayCfg.Provider.GetString(model.NameType))
	}
	service := &Service{
		GatewayConfig: gatewayCfg,
		provider:      provider,
		ctx:           context.TODO(),
	}
	return service, nil
}

// Start gateway service
func (s *Service) Start() error {
	zap.L().Debug("Starting a provider service", zap.String("name", s.GatewayConfig.Name), zap.String("provider", s.GatewayConfig.Provider.GetString(model.NameType)))

	// update topics
	s.topicListenFromCore = mcbus.GetTopicPostMessageToProvider(s.GatewayConfig.ID)
	s.topicPostToCore = mcbus.GetTopicPostMessageToCore()

	s.messageQueue = utils.GetQueue(fmt.Sprintf("queue_provider_message_%s", s.GatewayConfig.ID), messageQueueSize)
	s.rawMessageQueue = utils.GetQueue(fmt.Sprintf("queue_provider_raw_message_%s", s.GatewayConfig.ID), rawMessageQueueSize)
	s.sleepingMessageQueue = make(map[string][]*msgml.Message)

	// start handlers
	s.startMessageListener()

	// start raw message processor
	s.startRawMessageProcessor()

	// start provider
	err := s.provider.Start(s.addRawMessageToQueueFunc)
	if err != nil {
		zap.L().Error("Error", zap.Error(err))
		mcbus.Unsubscribe(s.topicListenFromCore, s.topicListenFromCoreSubscriptionID)

		s.messageQueue.Stop()
		s.rawMessageQueue.Stop()
	}
	return err
}

// unsubscribe and stop queues
func (s *Service) stopService() {
	mcbus.Unsubscribe(s.topicListenFromCore, s.topicListenFromCoreSubscriptionID)
	s.messageQueue.Stop()
	s.rawMessageQueue.Stop()
}

// Stop the service
func (s *Service) Stop() error {
	defer s.stopService()     // in any case when exit, call stopService
	err := s.provider.Close() // close protocol connection
	if err != nil {
		return err
	}
	return nil
}

// this function supplied to protocol
// rawMessages will be added directely here
func (s *Service) addRawMessageToQueueFunc(rawMsg *msgml.RawMessage) error {
	status := s.rawMessageQueue.Produce(rawMsg)
	if !status {
		return errors.New("Failed to add rawmessage in to queue")
	}
	return nil
}

// listens messages from core componenet
func (s *Service) startMessageListener() {
	subscriptionID, err := mcbus.Subscribe(s.topicListenFromCore, func(event *busml.Event) {
		msg, ok := event.Data.(*msgml.Message)
		if !ok {
			// convert bytes to struct
			bytes, isBytes := event.Data.([]byte)
			if !isBytes {
				zap.L().Warn("Received invalid type", zap.Any("event", event))
				return
			}
			message := &msgml.Message{}
			err := utils.ByteToStruct(bytes, message)
			if err != nil {
				zap.L().Warn("Failed to convet to target type", zap.Error(err))
				return
			}
			msg = message
		}

		if msg != nil {
			s.messageQueue.Produce(msg)
		}
	},
	)
	if err != nil {
		zap.L().Error("Failed to subscribe", zap.String("topic", s.topicListenFromCore), zap.Error(err))
	} else {
		s.topicListenFromCoreSubscriptionID = subscriptionID
	}
	s.messageQueue.StartConsumers(numberOfWorkers, s.messageConsumer)
}

func (s *Service) messageConsumer(item interface{}) {
	msg := item.(*msgml.Message)
	// for sleeping node message put it on sleeping queue and exit
	if msg.IsPassiveNode {
		s.addToSleepingMessageQueue(msg)
		return
	}

	// post message to protocol
	s.post(msg)
}

// converts and post the message to protocol
func (s *Service) post(msg *msgml.Message) {
	rawMsg, err := s.provider.ToRawMessage(msg)
	if err != nil {
		zap.L().Warn("Failed to parse", zap.String("gateway", s.GatewayConfig.ID), zap.Any("message", msg), zap.Error(err))
		return
	}
	err = s.provider.Post(rawMsg)
	if err != nil {
		zap.L().Warn("Failed to send", zap.String("gateway", s.GatewayConfig.ID), zap.Any("message", msg), zap.Any("rawMessage", rawMsg), zap.Error(err))
	}
}

// add message in to sleeping queue
func (s *Service) addToSleepingMessageQueue(msg *msgml.Message) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// add into sleeping queue
	queue, ok := s.sleepingMessageQueue[msg.NodeID]
	if !ok {
		queue = make([]*msgml.Message, 0)
		s.sleepingMessageQueue[msg.NodeID] = queue
	}
	queue = append(queue, msg)
	// if queue size exceeds maximum defined size, do resize
	oldSize := len(queue)
	if oldSize > sleepingQueueLimit {
		queue = queue[:sleepingQueueLimit]
		zap.L().Debug("Dropped messags from sleeping queue", zap.Int("oldSize", oldSize), zap.Int("newSize", len(queue)))
	}
}

// process received raw messages from protocol
func (s *Service) startRawMessageProcessor() {
	s.rawMessageQueue.StartConsumers(rawMessageNumberOfWorkers, func(data interface{}) {
		rawMsg := data.(*msgml.RawMessage)
		zap.L().Debug("RawMessage received", zap.String("gateway", s.GatewayConfig.Name), zap.Any("rawMessage", rawMsg))
		messages, err := s.provider.ToMessage(rawMsg)
		if err != nil {
			zap.L().Warn("Failed to parse", zap.String("gateway", s.GatewayConfig.ID), zap.Any("rawMessage", rawMsg), zap.Error(err))
			return
		}
		if len(messages) == 0 {
			zap.L().Debug("Messages not parsed", zap.String("gateway", s.GatewayConfig.Name), zap.Any("RawMessage", rawMsg))
			return
		}
		// update gatewayID if not found
		for index := 0; index < len(messages); index++ {
			msg := messages[index]
			if msg != nil {
				if msg != nil && msg.GatewayID == "" {
					msg.GatewayID = s.GatewayConfig.ID
				}
				err = mcbus.Publish(s.topicPostToCore, msg)
				if err != nil {
					zap.L().Debug("Messages failed to post on topic", zap.String("topic", s.topicPostToCore), zap.String("gateway", s.GatewayConfig.Name), zap.Any("message", msg), zap.Error(err))
					return
				}
			}
		}
	})
}

// ClearSleepingMessageQueue clears all the messages on the queue
func (s *Service) clearSleepingMessageQueue() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sleepingMessageQueue = make(map[string][]*msgml.Message)
}
