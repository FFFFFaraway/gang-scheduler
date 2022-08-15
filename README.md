# simple-gang-scheduler

[![Go](https://github.com/FFFFFaraway/gang-scheduler/actions/workflows/go.yml/badge.svg)](https://github.com/FFFFFaraway/gang-scheduler/actions/workflows/go.yml)

[![Go Report Card](https://goreportcard.com/badge/github.com/angao/scheduler-framework-sample)](https://goreportcard.com/report/github.com/angao/scheduler-framework-sample)

This repo is a simple gang scheduler implemented by scheduler framework in Kubernetes. This scheduler have a `sample` plugin, and implements `queue sort` and `permit` extension points. More information can be found in [this blog](https://fffffaraway.github.io/2022/08/14/利用Scheduling-Framework实现一个简单的gang调度器/).

## Install

```bash
git clone git@github.com:FFFFFaraway/gang-scheduler.git
cd gang-scheduler
# create rbac
kubectl apply -f deploy/rbac.yaml
# create sceduler
kubectl apply -f deploy/deployment.yaml
```

You can change the scheduler name by edit the `KubeSchedulerConfiguration` in `deploy/deployment.yaml` file. Default name is `gang-scheduler`, and the default namespace is `kube-system`.

## How to use

The scheduler use configmap to record podgroup information. For example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  # PodGroup name
  name: pending-pg
  namespace: sw
data:
  # Minimum number of allocated pods belongs to this PodGroup
  minAvailable: "3"
  # Wait seconds in scheduler queue
  scheduleTimeoutSeconds: "5"
```

Pod use label to bind pod group. For example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pending-example
  # The namespace should be the same as the PodGroup it belongs to
  namespace: sw
spec:
  replicas: 2
  selector:
    matchLabels:
      app: busybox
  template:
    metadata:
      labels:
        app: busybox
        # This label bind this pod to the 'pending-pg' PodGroup
        pod-group.scheduling.bdap.com/podgroup-configmap: pending-pg
    spec:
      # use our sheduler
      schedulerName: gang-scheduler
      containers:
      - image: busybox
        name: busybox
        command: ["sleep", "infinity"]
```

More examples can be found in `config` directory.
