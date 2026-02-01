# Pod Cleaner

## Features

### Basic

- Running in Kubernetes
- Could get pods in any ns, need to excludive specific ns("kube-system")
- Pods is necessary(determine by user, for example: "running" and "init" status are necessary) or not, restart(delete) unnecessary pods
- Cron job, runs every 10min, run until previous finishes
- Log all deleted pods

### Bonus

- If new pods still has problem after restart, send notification with error details of the pods.(Demo for sender part)
- In a large cluster, how to avoid performance problem? How to run faster enough to fit the 10min interval?

## Steps

1. Get pods and status(with pod.Status.Phase)
2. Is necessary? Store unnecessary pods for deletion and report
3. Delete unnecessary pod. Most time-consuming part. But in a real env, usually there not too much pod need to be deleted.
4. Monitor new pods, if new pods are unnecessary, send notification 

## Quick Start
The pod cleaner can be run as a CronJob, or Deployment.

### Run As A CronJob

```shell
kubectl create -f kubernetes/base/rbac.yaml
kubectl create -f kubernetes/base/configmap.yaml
kubectl create -f kubernetes/cronjob.yaml
```

### Run As A Deployment

```shell
kubectl create -f kubernetes/base/rbac.yaml
kubectl create -f kubernetes/base/configmap.yaml
kubectl create -f kubernetes/deployment.yaml
```

## Architecture

### Idea1: CronJob

Get pod from API-Server directly, need **pagination**

Heavy load to API-Server and ETCD in a large cluster

Set concurrencyPolicy for CronJob

### Idea2: Customized CronJob + CRD ❌

Extend CronJob with more spec field for requirements

### Idea3: Deployment(Controller)

Seprate delete action by event-trigger

Typical controller:

- Managed resources: unnecessary pods
- Actions: delete

Difference: report need to be separated(interval run)

### Idea4: CronJob/Deployment, with same container implementation ☑️

Runing once or multi times with flag `--cleaning-interval`

Using `Informer` to get pod list is more efficient and safer, it's not necessary to implement a different contaienr for CronJob

Getting new pods by using old pods' labels

Delete pod serially is satiable for current situation

## Local test

Total 2000, delete 1000. Take 3m, memory 1ess than 50Mi

Total 2000, delete 200. Take 40s, memory 1ess than 50Mi

## Potential optimization

1. Concurrent deletion, need to increase client QPS limit(default is 5)
2. Aggregate notification info, group by pod owner and status

## Reference

- [sample-controller](https://github.com/kubernetes/sample-controller)
- [kubernetes-sigs/descheduler](https://github.com/kubernetes-sigs/descheduler)
