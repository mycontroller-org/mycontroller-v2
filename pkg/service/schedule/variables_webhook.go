package schedule

import (
	"net/http"

	"github.com/mycontroller-org/server/v2/pkg/json"
	"github.com/mycontroller-org/server/v2/pkg/model"
	scheduleML "github.com/mycontroller-org/server/v2/pkg/model/schedule"
	httpclient "github.com/mycontroller-org/server/v2/pkg/utils/http_client_json"
	"go.uber.org/zap"
)

func loadWebhookVariables(scheduleID string, config scheduleML.CustomVariableConfig, variables map[string]interface{}) map[string]interface{} {
	whCfg := config.Webhook
	client := httpclient.GetClient(whCfg.InsecureSkipVerify)
	if !whCfg.IncludeConfig {
		delete(variables, model.KeySchedule)
	}

	if whCfg.Method == "" {
		whCfg.Method = http.MethodPost
	}

	res, resBody, err := client.Request(whCfg.URL, whCfg.Method, whCfg.Headers, whCfg.QueryParameters, variables, 0)
	responseStatusCode := 0
	if res != nil {
		responseStatusCode = res.StatusCode
	}
	if err != nil {
		zap.L().Error("error on executing webhook", zap.Error(err), zap.String("scheduleID", scheduleID), zap.String("url", whCfg.URL), zap.Int("responseStatusCode", responseStatusCode))
		return nil
	}

	resultMap := make(map[string]interface{})

	err = json.Unmarshal(resBody, &resultMap)
	if err != nil {
		zap.L().Error("error on converting to json", zap.Error(err), zap.String("response", string(resBody)))
		return nil
	}

	return resultMap
}
