/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sample

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	framework_rt "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	"testing"
)

const (
	queueSortPlugin = "no-op-queue-sort-plugin"
	bindPlugin      = "bind-plugin"
)

func newFrameworkWithQueueSortAndBind(r framework_rt.Registry, pl *config.Plugins, plc []config.PluginConfig, opts ...framework_rt.Option) (framework.Framework, error) {
	if _, ok := r[queueSortPlugin]; !ok {
		r[queueSortPlugin] = newQueueSortPlugin
	}
	if _, ok := r[bindPlugin]; !ok {
		r[bindPlugin] = newBindPlugin
	}
	plugins := &config.Plugins{}
	plugins.Append(pl)
	if len(plugins.QueueSort.Enabled) == 0 {
		plugins.Append(&config.Plugins{
			QueueSort: config.PluginSet{
				Enabled: []config.Plugin{{Name: queueSortPlugin}},
			},
		})
	}
	if len(plugins.Bind.Enabled) == 0 {
		plugins.Append(&config.Plugins{
			Bind: config.PluginSet{
				Enabled: []config.Plugin{{Name: bindPlugin}},
			},
		})
	}
	profile := &config.KubeSchedulerProfile{
		SchedulerName: "Something",
		Plugins:       plugins,
		PluginConfig:  plc,
	}
	return framework_rt.NewFramework(r, profile, opts...)
}

func TestPermit(t *testing.T) {
	tests := []struct {
		name     string
		pods     []*corev1.Pod
		expected []framework.Code
	}{
		{
			name: "common pod not belongs any podGroup",
			pods: []*corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod2"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod3"}},
			},
			expected: []framework.Code{framework.Success, framework.Success, framework.Success},
		},
		{
			name: "pods belongs podGroup",
			pods: []*corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Labels: map[string]string{PodGroupName: "pg1"}, UID: types.UID("pod1")}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Labels: map[string]string{PodGroupName: "pg1"}, UID: types.UID("pod2")}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod3", Labels: map[string]string{PodGroupName: "pg2"}, UID: types.UID("pod3")}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod4", Labels: map[string]string{PodGroupName: "pg1"}, UID: types.UID("pod4")}},
			},
			expected: []framework.Code{framework.Wait, framework.Wait, framework.Success, framework.Success},
		},
	}
	Name := "SomethingHere"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := framework_rt.Registry{}
			cfgPls := &config.Plugins{Permit: config.PluginSet{}}
			if err := registry.Register(Name,
				func(rt runtime.Object, handle framework.Handle) (framework.Plugin, error) {
					return &Sample{handle: handle, podLister: &fakePodLister{}, cmLister: &fakeConfigMapLister{}}, nil
				}); err != nil {
				t.Fatalf("fail to register filter plugin (%s)", Name)
			}

			// append plugins to permit pluginset
			cfgPls.Permit.Enabled = append(
				cfgPls.Permit.Enabled,
				config.Plugin{Name: Name})

			f, err := newFrameworkWithQueueSortAndBind(registry, cfgPls, emptyArgs, framework_rt.WithSnapshotSharedLister(&fakeSharedLister{}))
			if err != nil {
				t.Fatalf("fail to create framework: %s", err)
			}

			if got := f.RunPermitPlugins(context.TODO(), nil, tt.pods[0], ""); got.Code() != tt.expected[0] {
				t.Errorf("expected %v, got %v", tt.expected[0], got.Code())
			}
			if got := f.RunPermitPlugins(context.TODO(), nil, tt.pods[1], ""); got.Code() != tt.expected[1] {
				t.Errorf("expected %v, got %v", tt.expected[1], got.Code())
			}
			if got := f.RunPermitPlugins(context.TODO(), nil, tt.pods[2], ""); got.Code() != tt.expected[2] {
				t.Errorf("expected %v, got %v", tt.expected[2], got.Code())
			}
		})
	}
}
