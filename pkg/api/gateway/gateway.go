package gateway

import (
	ml "github.com/mycontroller-org/backend/pkg/model"
	gwml "github.com/mycontroller-org/backend/pkg/model/gateway"
	svc "github.com/mycontroller-org/backend/pkg/service"
	ut "github.com/mycontroller-org/backend/pkg/util"
)

// ListGateways by filter and pagination
func ListGateways(f []ml.Filter, p ml.Pagination) ([]gwml.Config, error) {
	out := make([]gwml.Config, 0)
	svc.STG.Find(ml.EntityGateway, f, p, &out)
	return out, nil
}

// GetGateway returns a gateway
func GetGateway(f []ml.Filter) (gwml.Config, error) {
	out := gwml.Config{}
	err := svc.STG.FindOne(ml.EntityGateway, f, &out)
	return out, err
}

// Save gateway config into disk
func Save(g *gwml.Config) error {
	if g.ID == "" {
		g.ID = ut.RandID()
	}
	return svc.STG.Upsert(ml.EntityGateway, nil, g)
}

// SetState Updates state data
func SetState(g *gwml.Config, s ml.State) error {
	g.State = s
	return Save(g)
}
