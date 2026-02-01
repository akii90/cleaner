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

// newPodAge is a standard for new pod
const newPodAge = 10 * time.Minute

type PodCleaner struct {
	kubeclientset kubernetes.Interface
	podLister     corelisters.PodLister
	podsSynced    cache.InformerSynced
	config        *config.PolicyConfig
	interval      time.Duration
	notifier      NotificationSender
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
		notifier:      NewDemoSender(),
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

	deletedCount := 0
	var deletedPods []string
	var deletedPodObjects []*corev1.Pod

	for _, pod := range pods {
		if c.isExcludedNamespaces(pod) {
			continue
		}
		if c.isExcludeStatus(pod) {
			continue
		}

		// Action: Delete
		logger.Info("Found Unhealthy Pod",
			"namespace", pod.Namespace,
			"name", pod.Name,
			"status", pod.Status.Phase)

		// Skip not existed pod
		if _, err := c.podLister.Pods(pod.Namespace).Get(pod.Name); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
		}

		err := c.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(err, "Failed to delete pod", "namespace", pod.Namespace, "name", pod.Name)
		} else {
			deletedCount++
			deletedPods = append(deletedPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			deletedPodObjects = append(deletedPodObjects, pod) // Store for verification
			logger.Info("Deleted pod", "namespace", pod.Namespace, "name", pod.Name)
		}
	}

	duration := time.Since(startTime)
	logger.Info("Clean Process Finished", "duration", duration, "deleted", deletedCount)
	if len(deletedPods) > 0 {
		logger.Info("Deleted Pods Summary", "pods", deletedPods)

		// Verify restarted pods
		checkDelay := time.Duration(c.config.CheckDelaySeconds) * time.Second
		logger.Info("Waiting for pods to restart...", "delay", checkDelay)

		select {
		case <-time.After(checkDelay):
			c.verifyRestarts(ctx, deletedPodObjects)
		case <-ctx.Done():
			logger.Info("Context cancelled before verify restarted pods")
		}
	}
}

func (c *PodCleaner) verifyRestarts(ctx context.Context, oldPods []*corev1.Pod) {
	logger := klog.FromContext(ctx)
	logger.Info("Verifying restarted pods...")

	uniquePods := make(map[string]*corev1.Pod)

	for _, oldPod := range oldPods {
		if len(oldPod.Labels) == 0 {
			continue
		}

		// List pods with same labels
		selector := labels.Set(oldPod.Labels).AsSelector()
		pods, err := c.podLister.Pods(oldPod.Namespace).List(selector)
		if err != nil {
			logger.Error(err, "Failed to list pods for verification", "namespace", oldPod.Namespace, "labels", oldPod.Labels)
			continue
		}

		for _, p := range pods {
			uniquePods[string(p.UID)] = p
		}
	}

	c.checkNewPods(ctx, uniquePods)
}

func (c *PodCleaner) checkNewPods(ctx context.Context, newPods map[string]*corev1.Pod) {
	logger := klog.FromContext(ctx)
	for _, p := range newPods {
		// Check if pod is "new" and "unNecessary"
		if p.Status.StartTime != nil {
			age := time.Since(p.Status.StartTime.Time)
			if age < newPodAge {
				if !c.isExcludeStatus(p) {
					msg := buildNotificationMessage(p)
					if err := c.notifier.Send(ctx, msg); err != nil {
						logger.Error(err, "Failed to send notification", "pod", fmt.Sprintf("%s/%s", p.Namespace, p.Name))
					}
				}
			}
		}
	}
}

func (c *PodCleaner) isExcludedNamespaces(pod *corev1.Pod) bool {
	for _, ns := range c.config.ExcludeNamespaces {
		if pod.Namespace == ns {
			return true
		}
	}
	return false
}

func (c *PodCleaner) isExcludeStatus(pod *corev1.Pod) bool {
	// Check pod status
	phaseMatch := false
	for _, status := range c.config.ExcludePodStatus {
		if strings.EqualFold(string(pod.Status.Phase), status) {
			phaseMatch = true
			break
		}
	}
	return phaseMatch
}
