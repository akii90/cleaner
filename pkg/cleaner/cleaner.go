package cleaner

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"strings"
	"time"

	"github.com/akii90/cleaner/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

type PodCleaner struct {
	kubeclientset kubernetes.Interface
	podLister     corelisters.PodLister
	podsSynced    cache.InformerSynced
	config        *config.PolicyConfig
	interval      time.Duration
}

func NewPodCleaner(
	client kubernetes.Interface,
	podInformer coreinformers.PodInformer,
	conf *config.PolicyConfig,
	interval time.Duration) *PodCleaner {

	return &PodCleaner{
		kubeclientset: client,
		podLister:     podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		config:        conf,
		interval:      interval,
	}
}

func (c *PodCleaner) Run(ctx context.Context) error {
	logger := klog.FromContext(ctx)
	logger.Info("Starting cleaner")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	if c.interval == 0 {
		logger.Info("Mode: One-Shot")
		c.clean(ctx)
	} else {
		logger.Info("Mode: Interval Loop", "interval", c.interval)
		wait.UntilWithContext(ctx, c.clean, c.interval)
	}

	return nil
}

func (c *PodCleaner) clean(ctx context.Context) {
	logger := klog.FromContext(ctx)
	logger.V(3).Info("Starting Cleaning")
	startTime := time.Now()

	// TODO: reduce memory usage
	pods, err := c.podLister.List(labels.Everything())
	if err != nil {
		logger.Error(err, "Error listing pods")
		return
	}

	processedCount := 0
	deletedCount := 0

	for _, pod := range pods {
		processedCount++
		if c.isExcluded(pod) {
			continue
		}
		if c.isHealthy(pod) {
			continue
		}

		// Action: Delete
		logger.Info("Found Unhealthy Pod",
			"namespace", pod.ObjectMeta.Namespace,
			"name", pod.ObjectMeta.Name,
			"status", pod.Status.Phase)

		// Skip not existed pod
		if _, err := c.podLister.Pods(pod.ObjectMeta.Namespace).Get(pod.ObjectMeta.Name); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
		}

		err := c.kubeclientset.CoreV1().Pods(pod.ObjectMeta.Namespace).Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(err, "Failed to delete pod", "namespace", pod.ObjectMeta.Namespace, "name", pod.ObjectMeta.Name)
		} else {
			deletedCount++
			logger.Info("Deleted pod", "namespace", pod.ObjectMeta.Namespace, "name", pod.ObjectMeta.Name)
		}
	}

	duration := time.Since(startTime)
	logger.Info("Cycle Finished", "duration", duration, "processed", processedCount, "deleted", deletedCount)
}

func (c *PodCleaner) isExcluded(pod *corev1.Pod) bool {
	for _, ns := range c.config.ExcludeNamespaces {
		if pod.ObjectMeta.Namespace == ns {
			return true
		}
	}
	return false
}

func (c *PodCleaner) isHealthy(pod *corev1.Pod) bool {
	// Check status
	phaseMatch := false
	for _, status := range c.config.HealthyStatus {
		if strings.EqualFold(string(pod.Status.Phase), status) {
			phaseMatch = true
			break
		}
	}
	return phaseMatch
}
