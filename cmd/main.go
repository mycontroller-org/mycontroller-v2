package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/mycontroller-org/mycontroller/cmd/app/handler"
	gwAPI "github.com/mycontroller-org/mycontroller/pkg/api/gateway"
	msgPRO "github.com/mycontroller-org/mycontroller/pkg/engine/message"
	srv "github.com/mycontroller-org/mycontroller/pkg/service"
)

func init() {
	start := time.Now()
	srv.Init()
	zap.L().Debug("Init complete", zap.String("timeTaken", time.Since(start).String()))
}

/*
func testQueue() {
	cfg := map[string]string{
		"url": "tcp://enveedu.mycontroller.org:2883",
	}
	c, err := mt.New(cfg)
	if err != nil {
		zap.L().Error("Error on creating client", zap.Error(err))
	}
	c.Subscribe("out_rfm69/#")
	c.Subscribe("in_rfm69/#")
}
*/
func main() {
	defer zap.L().Sync()
	// call shutdown handler
	go handleShutdown()

	// start engine
	msgPRO.Init()

	// load gateways
	start := time.Now()
	gwAPI.LoadGateways()
	zap.L().Debug("Load gateways done.", zap.String("timeTaken", time.Since(start).String()))

	err := handler.StartHandler()
	if err != nil {
		zap.L().Fatal("Error on starting http handler", zap.Error(err))
	}
}

func handleShutdown() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// waiting for signal
	sig := <-sigs
	close(sigs)

	start := time.Now()
	zap.L().Info("Shutdown initiated..", zap.Any("signal", sig))

	// unload gateways
	gwAPI.UnloadGateways()
	zap.L().Debug("Unload gateways done")

	// stop engine
	msgPRO.Close()

	// close services
	err := srv.Close()
	if err != nil {
		zap.L().Fatal("Error on closing services", zap.Error(err))
	}
	zap.L().Debug("Close services done")
	zap.L().Debug("All services closed", zap.String("timeTaken", time.Since(start).String()))
	zap.L().Debug("Bye, See you soon :)")

	// stop web/api service
	os.Exit(0)

}
