package action

import (
	fieldAPI "github.com/mycontroller-org/server/v2/pkg/api/field"
	"github.com/mycontroller-org/server/v2/pkg/model"
	msgML "github.com/mycontroller-org/server/v2/pkg/model/message"
	converterUtils "github.com/mycontroller-org/server/v2/pkg/utils/convertor"
	quickIdUtils "github.com/mycontroller-org/server/v2/pkg/utils/quick_id"
	stgType "github.com/mycontroller-org/server/v2/plugin/database/storage/type"
)

// ToFieldByID sends the payload to the given field
func ToFieldByID(id string, payload string) error {
	filters := []stgType.Filter{{Key: model.KeyID, Value: id}}
	field, err := fieldAPI.Get(filters)
	if err != nil {
		return err
	}
	return ToField(field.GatewayID, field.NodeID, field.SourceID, field.FieldID, payload)
}

// ToFieldByQuickID sends the payload to the given field
func ToFieldByQuickID(quickID string, payload string) error {
	_, idsMap, err := quickIdUtils.EntityKeyValueMap(quickID)
	if err != nil {
		return err
	}

	// really needs to check these ids on internal database?
	field, err := fieldAPI.GetByIDs(idsMap[model.KeyGatewayID], idsMap[model.KeyNodeID], idsMap[model.KeySourceID], idsMap[model.KeyFieldID])
	if err != nil {
		return err
	}
	return ToField(field.GatewayID, field.NodeID, field.SourceID, field.FieldID, payload)
}

// ToField sends the payload to the given ids
func ToField(gatewayID, nodeID, sourceID, fieldID, payload string) error {
	if payload == model.ActionToggle {
		// get field current data
		field, err := fieldAPI.GetByIDs(gatewayID, nodeID, sourceID, fieldID)
		if err != nil {
			return err
		}

		if converterUtils.ToBool(field.Current.Value) {
			payload = "false"
		} else {
			payload = "true"
		}
	}

	msg := msgML.NewMessage(false)
	msg.GatewayID = gatewayID
	msg.NodeID = nodeID
	msg.SourceID = sourceID
	pl := msgML.NewPayload()
	pl.Key = fieldID
	pl.Value = payload
	msg.Payloads = append(msg.Payloads, pl)
	msg.Type = msgML.TypeSet
	return Post(&msg)
}
