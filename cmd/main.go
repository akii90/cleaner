package main

import (
	"flag"
	"github.com/akii90/cleaner/pkg/cleaner"
	"github.com/akii90/cleaner/pkg/config"
	"github.com/akii90/cleaner/pkg/signals"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"time"

	"k8s.io/klog/v2"
)

var (
	kubeconfig       string
	masterURL        string
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

	// masterURL is used to overwriting api-server url in kubeconfig
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// kubeClient for generic Kubernetes APIs
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Hour*24)

	// ctx, clientSet, informer
	pc := cleaner.NewPodCleaner(kubeClient, kubeInformerFactory.Core().V1().Pods(), policy, cleaningInterval)

	// Start Informer
	logger.Info("Starting Informer...")
	kubeInformerFactory.Start(ctx.Done())

	// Run Cleaner
	if err = pc.Run(ctx); err != nil {
		logger.Error(err, "Error running cleaner")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&policyConfig, "policy-config", "", "Path to the policy configuration file (yaml)")
	flag.DurationVar(&cleaningInterval, "cleaning-interval", 0, "Interval for cleaning (e.g. 10m). If 0, runs once and exits.")
}
