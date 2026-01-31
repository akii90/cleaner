package main

import (
	"flag"
	"github.com/akii90/cleaner/pkg/config"
	"github.com/akii90/cleaner/pkg/signals"
	"time"

	"k8s.io/klog/v2"
)

var (
	kubeconfig       string
	policyConfig     string
	cleaningInterval time.Duration
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Set up signals so we handle the shutdown signal gracefully
	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	// Load Policy
	policy, err := config.LoadConfig(policyConfig)
	if err != nil {
		logger.Error(err, "Error loading policy config")
	}
	logger.Info("Pod cleaner policy conf",
		"healthyStatus", policy.HealthyStatus,
		"excludeNS", policy.ExcludeNamespaces,
		"checkDelay", policy.CheckDelaySeconds,
	)

}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&policyConfig, "policy-config", "", "Path to the policy configuration file (yaml)")
	flag.DurationVar(&cleaningInterval, "cleaning-interval", 0, "Interval for cleaning (e.g. 10m). If 0, runs once and exits.")
}
