package configuration

import (
	"flag"
	"fmt"
	"io/ioutil"

	cfgML "github.com/mycontroller-org/server/v2/pkg/model/config"
	"github.com/mycontroller-org/server/v2/pkg/utils/concurrency"
	loggerUtils "github.com/mycontroller-org/server/v2/pkg/utils/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// configuration globally accessable
var (
	CFG                   *cfgML.Config
	PauseModifiedOnUpdate = concurrency.SafeBool{}
)

// Load configuration
func Load() {
	// load a temporary logger
	logger := loggerUtils.GetLogger("development", "error", "console", false, 0)

	cf := flag.String("config", "./config.yaml", "Configuration file")
	flag.Parse()
	if cf == nil {
		logger.Fatal("configuration file not supplied")
		return
	}
	d, err := ioutil.ReadFile(*cf)
	if err != nil {
		logger.Fatal("error on reading configuration file", zap.Error(err))
	}

	err = yaml.Unmarshal(d, &CFG)
	if err != nil {
		logger.Fatal("failed to parse yaml data", zap.Error(err))
	}

	// update encryption key length
	// converts it to fixed size as 32 bytes
	CFG.Secret = updatedKey(CFG.Secret)

	// load default value
	PauseModifiedOnUpdate.Reset()
}

// UpdatedKey returns fixed key size
// that is 32 bytes
func updatedKey(actualKey string) string {
	return fmt.Sprintf("%032.32s", actualKey)
}
