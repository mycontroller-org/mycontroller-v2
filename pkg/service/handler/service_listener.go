package handler

import (
	busML "github.com/mycontroller-org/server/v2/pkg/model/bus"
	handlerType "github.com/mycontroller-org/server/v2/plugin/handler/type"
	rsML "github.com/mycontroller-org/server/v2/pkg/model/resource_service"
	sfML "github.com/mycontroller-org/server/v2/pkg/model/service_filter"
	"github.com/mycontroller-org/server/v2/pkg/service/mcbus"
	helper "github.com/mycontroller-org/server/v2/pkg/utils/filter_sort"
	queueUtils "github.com/mycontroller-org/server/v2/pkg/utils/queue"
	"go.uber.org/zap"
)

const (
	serviceMessageQueueLimit = 100
	serviceMessageQueueName  = "service_listener_handler"
)

var (
	serviceQueue *queueUtils.Queue
	svcFilter    *sfML.ServiceFilter
)

// Start handler service listener
func Start(filter *sfML.ServiceFilter) error {
	svcFilter = filter
	if svcFilter.Disabled {
		zap.L().Info("handler service disabled")
		return nil
	}

	if svcFilter.HasFilter() {
		zap.L().Info("handler service filter config", zap.Any("filter", svcFilter))
	} else {
		zap.L().Debug("there is no filter applied to handler service")
	}

	serviceQueue = queueUtils.New(serviceMessageQueueName, serviceMessageQueueLimit, postProcessServiceEvent, 1)

	// on message receive add it in to our local queue
	_, err := mcbus.Subscribe(mcbus.FormatTopic(mcbus.TopicServiceHandler), onServiceEvent)
	if err != nil {
		return err
	}

	err = initMessageListener()
	if err != nil {
		return err
	}

	// load handlers
	reqEvent := rsML.ServiceEvent{
		Type:    rsML.TypeHandler,
		Command: rsML.CommandLoadAll,
	}
	topicResourceServer := mcbus.FormatTopic(mcbus.TopicServiceResourceServer)
	return mcbus.Publish(topicResourceServer, reqEvent)
}

// Close the service listener
func Close() {
	if svcFilter.Disabled {
		return
	}
	UnloadAll()
	serviceQueue.Close()
	closeMessageListener()
}

func onServiceEvent(event *busML.BusData) {
	reqEvent := &rsML.ServiceEvent{}
	err := event.LoadData(reqEvent)
	if err != nil {
		zap.L().Warn("failed to convet to target type", zap.Error(err))
		return
	}
	if reqEvent == nil {
		zap.L().Warn("received a nil message", zap.Any("event", event))
		return
	}
	zap.L().Debug("event added into processing queue", zap.Any("event", reqEvent))
	status := serviceQueue.Produce(reqEvent)
	if !status {
		zap.L().Warn("failed to store the event into queue", zap.Any("event", reqEvent))
	}
}

// postProcessServiceEvent from the queue
func postProcessServiceEvent(event interface{}) {
	reqEvent := event.(*rsML.ServiceEvent)
	zap.L().Debug("processing a request", zap.Any("event", reqEvent))

	if reqEvent.Type != rsML.TypeHandler {
		zap.L().Warn("unsupported event type", zap.Any("event", reqEvent))
	}

	switch reqEvent.Command {
	case rsML.CommandStart:
		cfg := getConfig(reqEvent)
		if cfg != nil && helper.IsMine(svcFilter, cfg.Type, cfg.ID, cfg.Labels) {
			err := StartHandler(cfg)
			if err != nil {
				zap.L().Error("error on starting a handler", zap.Error(err), zap.String("handler", cfg.ID))
			}
		}

	case rsML.CommandStop:
		if reqEvent.ID != "" {
			err := StopHandler(reqEvent.ID)
			if err != nil {
				zap.L().Error("error on stopping a service", zap.Error(err))
			}
			return
		}
		cfg := getConfig(reqEvent)
		if cfg != nil {
			err := StopHandler(cfg.ID)
			if err != nil {
				zap.L().Error("error on stopping a service", zap.Error(err))
			}
		}

	case rsML.CommandReload:
		cfg := getConfig(reqEvent)
		if cfg != nil && helper.IsMine(svcFilter, cfg.Type, cfg.ID, cfg.Labels) {
			err := ReloadHandler(cfg)
			if err != nil {
				zap.L().Error("error on reload a service", zap.Error(err))
			}
		}

	case rsML.CommandUnloadAll:
		UnloadAll()

	default:
		zap.L().Warn("unsupported command", zap.Any("event", reqEvent))
	}
}

func getConfig(reqEvent *rsML.ServiceEvent) *handlerType.Config {
	cfg := &handlerType.Config{}
	err := reqEvent.LoadData(cfg)
	if err != nil {
		zap.L().Error("error on data conversion", zap.Any("data", reqEvent.Data), zap.Error(err))
		return nil
	}
	return cfg
}
