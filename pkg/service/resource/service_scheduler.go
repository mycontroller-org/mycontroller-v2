package resource

import (
	"errors"

	scheduleAPI "github.com/mycontroller-org/server/v2/pkg/api/schedule"
	rsML "github.com/mycontroller-org/server/v2/pkg/model/resource_service"
	scheduleML "github.com/mycontroller-org/server/v2/pkg/model/schedule"
	"go.uber.org/zap"
)

func schedulerService(reqEvent *rsML.ServiceEvent) error {
	resEvent := &rsML.ServiceEvent{
		Type:    reqEvent.Type,
		Command: reqEvent.ReplyCommand,
	}

	switch reqEvent.Command {
	case rsML.CommandGet:
		data, err := getScheduler(reqEvent)
		if err != nil {
			resEvent.Error = err.Error()
		}
		resEvent.SetData(data)

	case rsML.CommandUpdateState:
		err := updateSchedulerState(reqEvent)
		if err != nil {
			resEvent.Error = err.Error()
		}

	case rsML.CommandLoadAll:
		scheduleAPI.LoadAll()

	case rsML.CommandDisable:
		return disableScheduler(reqEvent)

	default:
		return errors.New("unknown command")
	}
	return postResponse(reqEvent.ReplyTopic, resEvent)
}

func getScheduler(request *rsML.ServiceEvent) (interface{}, error) {
	if request.ID != "" {
		cfg, err := scheduleAPI.GetByID(request.ID)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	} else if len(request.Labels) > 0 {
		filters := getLabelsFilter(request.Labels)
		result, err := scheduleAPI.List(filters, nil)
		if err != nil {
			return nil, err
		}
		return result.Data, nil
	}
	return nil, errors.New("filter not supplied")
}

func updateSchedulerState(reqEvent *rsML.ServiceEvent) error {
	if reqEvent.Data == nil {
		zap.L().Error("state not supplied", zap.Any("event", reqEvent))
		return errors.New("state not supplied")
	}

	state := &scheduleML.State{}
	err := reqEvent.LoadData(state)
	if err != nil {
		zap.L().Error("error on data conversion", zap.Any("data", reqEvent.Data), zap.Error(err))
		return err
	}

	return scheduleAPI.SetState(reqEvent.ID, state)
}

func disableScheduler(reqEvent *rsML.ServiceEvent) error {
	if reqEvent.Data == nil {
		zap.L().Error("scheduler id not supplied", zap.Any("event", reqEvent))
		return errors.New("id not supplied")
	}

	id := ""
	err := reqEvent.LoadData(&id)
	if err != nil {
		zap.L().Error("error on data conversion", zap.Any("reqEvent", reqEvent), zap.Error(err))
		return err
	}

	return scheduleAPI.Disable([]string{id})
}
