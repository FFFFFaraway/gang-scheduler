package sample

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

var emptyArgs = make([]config.PluginConfig, 0)

type fakePodLister struct {
	called int
}

// List lists all Pods in the indexer.
func (*fakePodLister) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	//TODO implement me
	panic("implement me")
}

// Pods returns an object that can list and get Pods.
func (*fakePodLister) Pods(namespace string) v1.PodNamespaceLister {
	return &fakePodNamespaceLister{}
}

type fakePodNamespaceLister struct {
}

func (*fakePodNamespaceLister) Get(name string) (*corev1.Pod, error) {
	//TODO implement me
	panic("implement me")
}

func (*fakePodNamespaceLister) List(selector labels.Selector) (ret []*corev1.Pod, err error) {
	// 假设目前集群中，Pod状态如下
	// namespace1:
	if selector.Matches(labels.Set{PodGroupName: "pg1"}) {
		ret = append(ret, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pg1-1", Namespace: "namespace1"}})
		return ret, nil
	}

	if selector.Matches(labels.Set{PodGroupName: "pg2"}) {
		ret = append(ret, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pg2-1", Namespace: "namespace2"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}})
		ret = append(ret, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pg2-2", Namespace: "namespace2"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}})
		ret = append(ret, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pg2-3", Namespace: "namespace2"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}})
		return ret, nil
	}

	return ret, nil
}

type fakeConfigMapLister struct {
	called int
}

func (l *fakeConfigMapLister) List(selector labels.Selector) (ret []*corev1.ConfigMap, err error) {
	//TODO implement me
	panic("implement me")
}

func (l *fakeConfigMapLister) ConfigMaps(namespace string) v1.ConfigMapNamespaceLister {
	return &fakeConfigMapNamespaceLister{}
}

type fakeConfigMapNamespaceLister struct {
}

func (l *fakeConfigMapNamespaceLister) List(selector labels.Selector) (ret []*corev1.ConfigMap, err error) {
	//TODO implement me
	panic("implement me")
}

// Get retrieves the ConfigMap from the indexer for a given namespace and name.
// Objects returned here must be treated as read-only.
func (l *fakeConfigMapNamespaceLister) Get(name string) (*corev1.ConfigMap, error) {
	if name == "pg1" {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "pg1", Namespace: "namespace1"},
			Data:       map[string]string{minAvailable: "3"},
		}, nil
	}
	if name == "pg2" {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "pg2", Namespace: "namespace2"},
			Data:       map[string]string{minAvailable: "3"},
		}, nil
	}
	return nil, errors.NewNotFound(corev1.Resource("configmap"), name)
}

type fakeSharedLister struct {
}

func (*fakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return nil
}
