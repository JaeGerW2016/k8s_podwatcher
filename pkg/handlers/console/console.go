package console

import (
	"k8s.io/klog"
	"k8s_podwatcher/pkg/handlers"
)

type handler struct {
}

func NewHandler() handlers.Handler {
	return (*handler)(nil)
}

func (*handler) Handle(event *handlers.Event) error {
	klog.Errorf("%v", event)
	return nil
}
