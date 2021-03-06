package forwardpayload

import (
	"github.com/mycontroller-org/server/v2/pkg/model"
	eventML "github.com/mycontroller-org/server/v2/pkg/model/bus/event"
	fwdPayloadML "github.com/mycontroller-org/server/v2/pkg/model/forward_payload"
	"github.com/mycontroller-org/server/v2/pkg/service/mcbus"
	stg "github.com/mycontroller-org/server/v2/pkg/service/database/storage"
	"github.com/mycontroller-org/server/v2/pkg/utils"
	busUtils "github.com/mycontroller-org/server/v2/pkg/utils/bus_utils"
	stgType "github.com/mycontroller-org/server/v2/plugin/database/storage/type"
)

// List by filter and pagination
func List(filters []stgType.Filter, pagination *stgType.Pagination) (*stgType.Result, error) {
	result := make([]fwdPayloadML.Config, 0)
	return stg.SVC.Find(model.EntityForwardPayload, &result, filters, pagination)
}

// Get returns a item
func Get(filters []stgType.Filter) (*fwdPayloadML.Config, error) {
	result := &fwdPayloadML.Config{}
	err := stg.SVC.FindOne(model.EntityForwardPayload, result, filters)
	return result, err
}

// Save a item details
func Save(fp *fwdPayloadML.Config) error {
	eventType := eventML.TypeUpdated
	if fp.ID == "" {
		fp.ID = utils.RandUUID()
		eventType = eventML.TypeCreated
	}
	filters := []stgType.Filter{
		{Key: model.KeyID, Value: fp.ID},
	}
	err := stg.SVC.Upsert(model.EntityForwardPayload, fp, filters)
	if err != nil {
		return err
	}
	busUtils.PostEvent(mcbus.TopicEventForwardPayload, eventType, model.EntityForwardPayload, fp)
	return nil
}

// Delete items
func Delete(IDs []string) (int64, error) {
	filters := []stgType.Filter{{Key: model.KeyID, Operator: stgType.OperatorIn, Value: IDs}}
	return stg.SVC.Delete(model.EntityForwardPayload, filters)
}

// Enable forward payload entries
func Enable(ids []string) error {
	filters := []stgType.Filter{{Key: model.KeyID, Operator: stgType.OperatorIn, Value: ids}}
	pagination := &stgType.Pagination{Limit: 100}
	response, err := List(filters, pagination)
	if err != nil {
		return err
	}

	mappings := *response.Data.(*[]fwdPayloadML.Config)
	for index := 0; index < len(mappings); index++ {
		mapping := mappings[index]
		if !mapping.Enabled {
			mapping.Enabled = true
			err = Save(&mapping)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Disable forward entries
func Disable(ids []string) error {
	filters := []stgType.Filter{{Key: model.KeyID, Operator: stgType.OperatorIn, Value: ids}}
	pagination := &stgType.Pagination{Limit: 100}
	response, err := List(filters, pagination)
	if err != nil {
		return err
	}
	mappings := *response.Data.(*[]fwdPayloadML.Config)
	for index := 0; index < len(mappings); index++ {
		mapping := mappings[index]
		if mapping.Enabled {
			mapping.Enabled = false
			err = Save(&mapping)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
