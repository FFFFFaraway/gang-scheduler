package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/FFFFFaraway/gang-scheduler/pkg/plugins/sample"
	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := app.NewSchedulerCommand(
		app.WithPlugin(sample.Name, sample.New),
	)

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
