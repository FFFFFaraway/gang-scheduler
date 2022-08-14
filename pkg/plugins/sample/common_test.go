package sample

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

var _ framework.QueueSortPlugin = &TestQueueSortPlugin{}

// TestQueueSortPlugin is a no-op implementation for QueueSort extension point.
type TestQueueSortPlugin struct{}

func newQueueSortPlugin(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &TestQueueSortPlugin{}, nil
}

func (pl *TestQueueSortPlugin) Name() string {
	return queueSortPlugin
}

func (pl *TestQueueSortPlugin) Less(_, _ *framework.QueuedPodInfo) bool {
	return false
}

var _ framework.BindPlugin = &TestBindPlugin{}

// TestBindPlugin is a no-op implementation for Bind extension point.
type TestBindPlugin struct{}

func newBindPlugin(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &TestBindPlugin{}, nil
}

func (t TestBindPlugin) Name() string {
	return bindPlugin
}

func (t TestBindPlugin) Bind(ctx context.Context, state *framework.CycleState, p *corev1.Pod, nodeName string) *framework.Status {
	return nil
}
