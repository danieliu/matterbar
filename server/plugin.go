package main

import (
	"sync"

	"github.com/mattermost/mattermost-server/plugin"
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
