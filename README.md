# Pod Cleaner

## Features

### Basic

- Running in Kubernetes
- Could get pods in any ns, need to excludive specific ns("kube-system")
- Pods is healthy(determine by user, for example: "running" and "init" status are healthy) or not, restart(delete) unhealthy pods
- Cron job, runs every 10min, run until previous finishes
- Log all deleted pods

### Bonus

- If new pods still has problem after restart, send notification with error details of the pods.(Demo for sender part)
- In a large cluster, how to avoid performance problem? How to run faster enough to fit the 10 min interval?

## Steps

1. Get pods status
2. IsHealthy? Store unhealthy pods
3. Delete unhealthy pod, log pods that be cleaned
4. Monitor new pods(less than 10m), if new pods are unhealthy, send notofication about

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

- Managed resources: unhealthy pods
- Actions: delete

Difference: report need to be separated(interval run)

### Idea4: CronJob/Deployment, with same container implementation ☑️

Runing once and multi times

Using `Informer` to get pod list is more efficient and safer, it's not necessary to implement a different contaienr for CronJob

## Reference

- [kubernetes-sigs/descheduler](https://github.com/kubernetes-sigs/descheduler)
