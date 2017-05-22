package main

import (
	"log"
	"os"
	"time"

	"github.com/jamiehannaford/canary-operator/pkg/controller"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	le "k8s.io/kubernetes/pkg/client/leaderelection"
	rl "k8s.io/kubernetes/pkg/client/leaderelection/resourcelock"
)

const (
	lockName = "canary-operator"
)

var (
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
)

func main() {
	ns := os.Getenv("NAMESPACE")
	if len(ns) == 0 {
		log.Fatal("NAMESPACE is a required env var")
	}

	id, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}

	client, err := kubeClient()
	if err != nil {
		log.Fatalf("Failed to generate client: %v", err)
	}

	config := rl.ResourceLockConfig{
		Identity:      id,
		EventRecorder: &record.FakeRecorder{},
	}
	lock, err := rl.New(rl.ConfigMapsResourceLock, ns, lockName, client, config)
	if err != nil {
		log.Fatalf("Failed to create lock: %v", err)
	}

	le.RunOrDie(le.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: le.LeaderCallbacks{
			OnStartedLeading: runApp,
			OnStoppedLeading: func() {
				log.Fatalf("Leader election lost")
			},
		},
	})
}

func kubeClient() (*clientset.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfigOrDie(cfg), nil
}

func runApp(stop <-chan struct{}) {
	for {
		c := controller.New()
		err := c.Run()
		switch err {
		default:
			log.Fatalf("Could not run controller: %v", err)
		}
	}
}
