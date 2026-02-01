package cleaner

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type NotificationMessage struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Phase     string `json:"phase"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}

// NotificationSender is the interface for sending notifications
type NotificationSender interface {
	Send(ctx context.Context, msg *NotificationMessage) error
}

// DemoSender is a pseudo implementation of NotificationSender
type DemoSender struct{}

func NewDemoSender() *DemoSender {
	return &DemoSender{}
}

func (s *DemoSender) Send(ctx context.Context, msg *NotificationMessage) error {
	klog.FromContext(ctx).Info("🔔 [Notification Sender] Sending Alert",
		"Namespace", msg.Namespace,
		"Pod", msg.PodName,
		"Phase", msg.Phase,
		"Reason", msg.Reason,
	)
	return nil
}

func buildNotificationMessage(pod *corev1.Pod) *NotificationMessage {
	return &NotificationMessage{
		Namespace: pod.Namespace,
		PodName:   pod.Name,
		Phase:     string(pod.Status.Phase),
		Reason:    pod.Status.Reason,
		Message:   pod.Status.Message,
	}
}
