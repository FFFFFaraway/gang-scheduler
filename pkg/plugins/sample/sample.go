package sample

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	// Name is plugin name
	Name         = "sample"
	PodGroupName = "pod-group.scheduling.bdap.com/name"
)

var _ framework.FilterPlugin = &Sample{}
var _ framework.PermitPlugin = &Sample{}

type Sample struct {
	handle    framework.Handle
	cmLister  clientv1.ConfigMapLister
	podLister clientv1.PodLister
}

func New(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	cmLister := handle.SharedInformerFactory().Core().V1().ConfigMaps().Lister()
	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
	return &Sample{
		handle:    handle,
		cmLister:  cmLister,
		podLister: podLister,
	}, nil
}

func (s *Sample) Name() string {
	return Name
}

func (s *Sample) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, node *framework.NodeInfo) *framework.Status {
	return framework.NewStatus(framework.Success, "")
}

func (s *Sample) Permit(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	podGroupName, exist := pod.Labels[PodGroupName]
	if !exist || podGroupName == "" {
		return framework.NewStatus(framework.Success, ""), 0
	}
	cm, err := s.cmLister.ConfigMaps(pod.Namespace).Get(podGroupName)
	if apierrors.IsNotFound(err) {
		klog.Errorf("podgroup %v configmap not found in %v", podGroupName, pod.Namespace)
		return framework.NewStatus(framework.Error, "podgroup configmap not found, please create configmap first"), 0
	}
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), 0
	}

	minAvailableStr, exist := cm.Data["minAvailable"]
	if !exist {
		return framework.NewStatus(framework.Error, "minAvailable field not found in podgroup configmap"), 0
	}
	minAvailable, err := strconv.Atoi(minAvailableStr)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), 0
	}
	if minAvailable <= 1 {
		return framework.NewStatus(framework.Success, ""), 0
	}

	scheduleTimeoutSeconds := 10
	scheduleTimeoutSecondsStr, exist := cm.Data["scheduleTimeoutSeconds"]
	if exist {
		scheduleTimeoutSeconds, err = strconv.Atoi(scheduleTimeoutSecondsStr)
		if err != nil {
			return framework.NewStatus(framework.Error, err.Error()), 0
		}
	}

	namespace := pod.Namespace

	running := 0
	selector := labels.Set{PodGroupName: podGroupName}.AsSelector()
	pods, err := s.podLister.Pods(namespace).List(selector)
	for _, p := range pods {
		if p.Status.Phase == v1.PodRunning {
			running++
		}
	}

	waiting := 0
	s.handle.IterateOverWaitingPods(func(wp framework.WaitingPod) {
		if wp.GetPod().Labels[PodGroupName] == podGroupName && wp.GetPod().Namespace == namespace {
			waiting++
		}
	})

	current := running + waiting + 1

	if current < minAvailable {
		msg := fmt.Sprintf("The count of podGroup %v/%v/%v is not up to minAvailable(%d) in Permit: running(%d), waiting(%d)",
			pod.Namespace, podGroupName, pod.Name, minAvailable, running, waiting)
		klog.V(3).Info(msg)
		return framework.NewStatus(framework.Wait, msg), time.Duration(scheduleTimeoutSeconds) * time.Second
	}

	klog.V(3).Infof("The count of podGroup %v/%v/%v is up to minAvailable(%d) in Permit: running(%d), waiting(%d)",
		pod.Namespace, podGroupName, pod.Name, minAvailable, running, waiting)
	s.handle.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
		if waitingPod.GetPod().Namespace == namespace && waitingPod.GetPod().Labels[PodGroupName] == podGroupName {
			klog.V(3).Infof("Permit allows the pod: %v/%v", podGroupName, waitingPod.GetPod().Name)
			waitingPod.Allow(s.Name())
		}
	})

	return framework.NewStatus(framework.Success, ""), 0
}
