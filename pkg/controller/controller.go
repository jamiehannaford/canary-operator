package controller

import (
	"fmt"
	"log"

	"github.com/jamiehannaford/canary-operator/pkg/canary"
	"github.com/jamiehannaford/canary-operator/pkg/spec"
	watch "k8s.io/apimachinery/pkg/watch"
)

type Config struct {
	Namespace string
}

type Controller struct {
	config Config

	canaries map[string]*canary.Canary
}

type Event struct {
	Type   watch.EventType
	Object *spec.Canary
}

func New(ns string) Controller {
	return Controller{
		config: Config{
			Namespace: ns,
		},
	}
}

func (c Controller) Run() error {
	// create watch/error channels
	eventCh, errorCh := c.watchCanaries()

	// handle any canary resource related event
	go func() {
		for event := range eventCh {
			if err := c.handleClusterEvent(event); err != nil {
				log.Fatal(err)
			}
		}
	}()

	// return any received errors immediately and break execution
	return <-errorCh
}

func (c Controller) watchCanaries() (<-chan *Event, <-chan error) {
	eventCh := make(chan *Event)
	errorCh := make(chan error)

	go func() {
		defer close(eventCh)

		for {

			// watch canaries

			// check errors

			// decode

			// create event

			// check errors

			// add event to channel

			// close body
		}
	}()

	return eventCh, errorCh
}

func (c Controller) handleClusterEvent(event *Event) error {
	canarySpec := event.Object
	canaryName := canarySpec.Metadata.Name

	switch event.Type {
	case watch.Added:
		newCanary := canary.New(canarySpec)
		c.canaries[canaryName] = newCanary

	case watch.Modified:
		if _, ok := c.canaries[canaryName]; !ok {
			return fmt.Errorf("unsafe state. canary was never created but we received event (%s)", event.Type)
		}
		c.canaries[canaryName].Update(canarySpec)

	case watch.Deleted:
		if _, ok := c.canaries[canaryName]; !ok {
			return fmt.Errorf("unsafe state. canary was never created but we received event (%s)", event.Type)
		}
		c.canaries[canaryName].Delete()
		delete(c.canaries, canaryName)

	}

	return nil
}
