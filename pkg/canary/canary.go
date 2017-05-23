package canary

import "github.com/jamiehannaford/canary-operator/pkg/spec"

type Canary struct {
}

func New(canarySpec *spec.Canary) *Canary {
	return &Canary{}
}

func (c *Canary) Update(canarySpec *spec.Canary) {

}

func (c *Canary) Delete() {

}
