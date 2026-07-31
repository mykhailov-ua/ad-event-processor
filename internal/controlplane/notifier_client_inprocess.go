package controlplane

import (
	"espx/internal/config"
	"espx/internal/notifier"
)

func NewNotifierClientInProcess(api notifier.NotifierAPI) *NotifierClient {
	return NewNotifierClientFromAPI(api)
}

func openNotifierClient(cfg *config.Config, opts ServeOptions) (*NotifierClient, error) {
	if opts.Notifier != nil {
		return opts.Notifier, nil
	}
	return NewNotifierClient(cfg)
}
