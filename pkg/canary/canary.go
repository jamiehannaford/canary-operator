package canary

import (
	"sync"
	"time"

	"github.com/jamiehannaford/canary-operator/pkg/spec"
	"k8s.io/client-go/kubernetes"
)

const (
	reconcileInterval = time.Second * 10
)

type Canary struct {
	config Config
	spec   *spec.Canary
}

type Config struct {
	KubeCli kubernetes.Interface
}

func New(config Config, canarySpec *spec.Canary, haltCh chan bool, wg *sync.WaitGroup) *Canary {
	c := &Canary{
		config: config,
		spec:   canarySpec,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.run(haltCh)
	}()

	return c
}

func (c *Canary) run(haltCh chan bool) {
	for {
		select {
		case <-haltCh:
			return
		case <-time.After(reconcileInterval):
			// main reconciliation loop
			return
		}
	}
}

func (c *Canary) Update(canarySpec *spec.Canary) {

}

func (c *Canary) Delete() {

}
