package main

import (
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	botUserName    = "rollbar"
	botDisplayName = "Rollbar"
	botDescription = "Rollbar->Mattermost webhook bot created by the Matterbar plugin."
)

type RollbarPlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// user ID of the bot account
	botUserID string
}

// OnActivate sets the bot user up
func (p *RollbarPlugin) OnActivate() error {
	botUserID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}
	p.botUserID = botUserID

	return nil
}
