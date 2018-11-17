package main

import (
	"sync"

	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const PluginId = "matterbar"

type RollbarPlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnDeactivate unregisters the command
func (p *RollbarPlugin) OnDeactivate() error {
	err := p.API.UnregisterCommand("", commandTrigger)
	if err != nil {
		return errors.Wrap(err, "failed to dectivate command")
	}
	return nil
}
