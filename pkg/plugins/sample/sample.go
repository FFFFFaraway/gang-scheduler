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
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	// Name is plugin name
	Name                   = "sample"
	PodGroupName           = "pod-group.scheduling.bdap.com/podgroup-configmap"
	minAvailable           = "minAvailable"
	scheduleTimeoutSeconds = "scheduleTimeoutSeconds"
)

var _ framework.QueueSortPlugin = &Sample{}
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

func (s *Sample) pgTime(p *v1.Pod) (time.Time, bool) {
	pg, exist := p.Labels[PodGroupName]
	if !exist || pg == "" {
		return time.Time{}, false
	}
	cm, err := s.cmLister.ConfigMaps(p.Namespace).Get(pg)
	if err != nil {
		return time.Time{}, false
	}
	return cm.CreationTimestamp.Time, true
}

func regPodLess(p1 *framework.QueuedPodInfo, p2 *framework.QueuedPodInfo) bool {
	prio1 := corev1helpers.PodPriority(p1.Pod)
	prio2 := corev1helpers.PodPriority(p2.Pod)
	if prio1 != prio2 {
		return prio1 > prio2
	}
	return p1.InitialAttemptTimestamp.Before(p2.InitialAttemptTimestamp)
}

func (s *Sample) Less(p1 *framework.QueuedPodInfo, p2 *framework.QueuedPodInfo) bool {
	pgt1, exist1 := s.pgTime(p1.Pod)
	pgt2, exist2 := s.pgTime(p1.Pod)
	// One is in pg while the other is not.
	// Then p1 first if p1 is not in a pod group
	if exist1 != exist2 {
		return !exist1
	}
	// Neither in pod group
	if !exist1 {
		return regPodLess(p1, p2)
	}
	if !pgt1.Equal(pgt2) {
		return pgt1.Before(pgt2)
	}
	return regPodLess(p1, p2)
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

	maStr, exist := cm.Data[minAvailable]
	if !exist {
		return framework.NewStatus(framework.Error, "minAvailable field not found in podgroup configmap"), 0
	}
	ma, err := strconv.Atoi(maStr)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error()), 0
	}
	if ma <= 1 {
		return framework.NewStatus(framework.Success, ""), 0
	}

	sts := 10
	stsStr, exist := cm.Data[scheduleTimeoutSeconds]
	if exist {
		sts, err = strconv.Atoi(stsStr)
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

	if current < ma {
		msg := fmt.Sprintf("The count of podGroup %v/%v/%v is not up to minAvailable(%d) in Permit: running(%d), waiting(%d)",
			pod.Namespace, podGroupName, pod.Name, ma, running, waiting)
		klog.V(3).Info(msg)
		return framework.NewStatus(framework.Wait, msg), time.Duration(sts) * time.Second
	}

	klog.V(3).Infof("The count of podGroup %v/%v/%v is up to minAvailable(%d) in Permit: running(%d), waiting(%d)",
		pod.Namespace, podGroupName, pod.Name, ma, running, waiting)
	s.handle.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
		if waitingPod.GetPod().Namespace == namespace && waitingPod.GetPod().Labels[PodGroupName] == podGroupName {
			klog.V(3).Infof("Permit allows the pod: %v/%v", podGroupName, waitingPod.GetPod().Name)
			waitingPod.Allow(s.Name())
		}
	})

	return framework.NewStatus(framework.Success, ""), 0
}
