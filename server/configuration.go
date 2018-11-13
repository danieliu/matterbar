package main

import (
	"fmt"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
type configuration struct {
	// (Required) The default team where Rollbar webhooks will be posted.
	DefaultTeam string

	// The default channel where Rollbar webhooks will be posted, created
	// automatically for the the default team if it doesn't exist. Defaults to
	// `Rollbar`.
	DefaultChannel string

	// The user that this plugin posts as, created automatically if it doesn't
	// exist. Defaults to `Rollbar`.
	Username string

	// The generated secret that will be used to authenticate incoming webhook
	// requests coming from Rollbar.
	Secret string

	// Corresponding ids of the above
	teamId    string
	channelId string
	userId    string
}

// Clone deep copies the configuration. Your implementation may only require a
// shallow copy if your configuration has no reference types.
func (c *configuration) Clone() *configuration {
	return &configuration{
		DefaultTeam:    c.DefaultTeam,
		DefaultChannel: c.DefaultChannel,
		Username:       c.Username,
		Secret:         c.Secret,
		teamId:         c.teamId,
		channelId:      c.channelId,
		userId:         c.userId,
	}
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
//
// Ensures the default team is configured, and that the default channel and user
// are created for use by the plugin.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)
	var err error

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	// validate configuration settings
	configuration.userId, err = p.ensureUserExists(configuration)
	if err != nil {
		return errors.Wrap(err, "failed to ensure user exists")
	}

	configuration.teamId, err = p.ensureDefaultTeamExists(configuration)
	if err != nil {
		return errors.Wrap(err, "failed to ensure default team exists")
	}

	configuration.channelId, err = p.ensureDefaultChannelExists(configuration)
	if err != nil {
		return errors.Wrap(err, "failed to ensure default channel exists")
	}

	p.setConfiguration(configuration)

	return nil
}

// Ensures the configured default user exists
func (p *Plugin) ensureUserExists(configuration *configuration) (string, error) {
	var err *model.AppError

	// Check for the configured user. Ignore any error, since it's hard to
	// distinguish runtime errors from a user simply not existing.
	user, _ := p.API.GetUserByUsername(configuration.Username)

	// Check that the configured user exists.
	if user == nil {
		// TODO: make sure the returned error is actually logged and somehow
		// visible to whoever is changing this setting, then remove the explicit
		// warn log call
		p.API.LogWarn(fmt.Sprintf("Configuration invalid: no user with Username %s exists",
			configuration.Username))
		return "", err
	}

	return user.Id, nil
}

// Ensures the configured default team exists
func (p *Plugin) ensureDefaultTeamExists(configuration *configuration) (string, error) {
	var err *model.AppError

	// If not configured, we can just expect a `team` query parameter in all
	// incoming http requests instead.
	if configuration.DefaultTeam == "" {
		return "", nil
	}

	team, _ := p.API.GetTeamByName(configuration.DefaultTeam)

	// Check that the configured team exists.
	if team == nil {
		// TODO: make sure the returned error is actually logged and somehow
		// visible to whoever is changing this setting, then remove the explicit
		// warn log call
		p.API.LogWarn(fmt.Sprintf("Configuration invalid: no team named %s exists",
			configuration.DefaultTeam))
		return "", err
	}

	return team.Id, nil
}

// Ensures the configured default channel exists on the configured team
func (p *Plugin) ensureDefaultChannelExists(configuration *configuration) (string, error) {
	// If not configured, we can just expect a `channel` query parameter in all
	// incoming http requests instead.
	if configuration.DefaultChannel == "" {
		return "", nil
	}

	// DefaultTeam must be set in order to have a DefaultChannel
	if configuration.DefaultTeam == "" && configuration.DefaultChannel != "" {
		// TODO: make sure the returned error is actually logged and somehow
		// visible to whoever is changing this setting, then remove the explicit
		// warn log call
		warnMessage := "Configuration invalid: a DefaultTeam must be specified before a DefaultChannel"
		p.API.LogWarn(warnMessage)
		return "", model.NewAppError(
			"ensureDefaultChannelExists",
			"matterbar.server.configuration.default_team_missing_for_default_channel",
			nil,
			warnMessage,
			400)
	}

	channel, _ := p.API.GetChannelByNameForTeamName(configuration.DefaultTeam,
		configuration.DefaultChannel,
		false)

	// Check that the configured channel exists.
	if channel == nil {
		// TODO: make sure the returned error is actually logged and somehow
		// visible to whoever is changing this setting, then remove the explicit
		// warn log call
		warnMessage := fmt.Sprintf(
			"Configuration invalid: no channel named %s exists",
			configuration.DefaultChannel)
		p.API.LogWarn(warnMessage)
		return "", model.NewAppError(
			"ensureDefaultChannelExists",
			"matterbar.server.configuration.default_channel_does_not_exist",
			nil,
			warnMessage,
			400)
	}

	return channel.Id, nil
}
