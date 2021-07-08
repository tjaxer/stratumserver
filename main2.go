package main

import (
	//"context"
	"context"
	"fmt"
	appctx "github.com/nixys/nxs-go-appctx/v2"
	"github.com/sirupsen/logrus"
	"github.com/tjaxer/stratumserver/ctx"
	"github.com/tjaxer/stratumserver/logger"
	server "github.com/tjaxer/stratumserver/routines/stratum-server"
	"os"
	"syscall"
)

func init() {
	logger.Init()
}

func main() {
	// Read command line arguments
	args := ctx.ArgsRead()

	// Init appctx
	appCtx, err := appctx.ContextInit(appctx.Settings{
		CustomContext:    &ctx.Ctx{},
		Args:             &args,
		CfgPath:          args.ConfigPath,
		TermSignals:      []os.Signal{syscall.SIGTERM, syscall.SIGINT},
		ReloadSignals:    []os.Signal{syscall.SIGHUP},
		LogrotateSignals: []os.Signal{syscall.SIGUSR1},
		LogFormatter:     &logrus.JSONFormatter{},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	appCtx.Log().SetLevel(logrus.TraceLevel)

	appCtx.Log().Info("program started")

	// main() body function
	defer appCtx.MainBodyGeneric()

	// Create main context
	c := context.Background()

	//// Create API server routine
	appCtx.RoutineCreate(c, server.Runtime)
	//
	//// Create users counter
	//appCtx.RoutineCreate(c, userscounter.Runtime)
}
