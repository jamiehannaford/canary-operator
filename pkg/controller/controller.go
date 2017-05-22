package controller

type Controller struct {
}

func New() Controller {
	return Controller{}
}

func (c Controller) Run() error {
	return nil
}
