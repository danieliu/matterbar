package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func TestCloningConfiguration(t *testing.T) {
	t.Run("cloning configuration", func(t *testing.T) {
		configuration := &configuration{
			DefaultTeam:    "team",
			DefaultChannel: "channel",
			Username:       "username",
			Secret:         "secret",
			teamId:         "teamId",
			channelId:      "channelId",
			userId:         "userId",
		}
		cloned := configuration.Clone()
		if &cloned == &configuration {
			t.Errorf("Configuration was not cloned.")
		}
		if *cloned != *configuration {
			t.Errorf("Cloned configuration different. %s != %s",
				cloned,
				configuration)
		}
	})
}

func TestGetSetConfiguration(t *testing.T) {
	t.Run("null configuration", func(t *testing.T) {
		plugin := &Plugin{}
		if configuration := plugin.getConfiguration(); configuration == nil {
			t.Error("Expected configuration to not be nil")
		}
	})

	t.Run("changing configuration", func(t *testing.T) {
		plugin := &Plugin{}
		configuration1 := &configuration{userId: "123"}
		plugin.setConfiguration(configuration1)
		if configuration1 != plugin.getConfiguration() {
			t.Errorf("Configurations not equal. %s != %s",
				configuration1,
				plugin.getConfiguration())
		}

		configuration2 := &configuration{userId: "456"}
		plugin.setConfiguration(configuration2)
		if configuration2 != plugin.getConfiguration() {
			t.Errorf("Configurations not equal. %s != %s",
				configuration2,
				plugin.getConfiguration())
		}
		if configuration1 == plugin.getConfiguration() {
			t.Errorf("Configuration did not change. %s",
				plugin.getConfiguration())
		}
	})

	t.Run("setting same configuration", func(t *testing.T) {
		plugin := &Plugin{}
		configuration1 := &configuration{}
		plugin.setConfiguration(configuration1)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Setting same configuration did not panic")
			}
		}()
		plugin.setConfiguration(configuration1)
	})

	t.Run("clearing configuration", func(t *testing.T) {
		plugin := &Plugin{}

		configuration1 := &configuration{teamId: "1"}
		plugin.setConfiguration(configuration1)
		defer func() {
			if r := recover(); r != nil {
				t.Error("Setting same configuration did not panic")
			}
		}()
		plugin.setConfiguration(nil)
		config := plugin.getConfiguration()
		if config == nil {
			t.Error("Configuration is nil")
		}
		if config == configuration1 {
			t.Error("Configuration did not change")
		}
	})
}

func TestOnConfigurationChange(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{
			Username:       "username",
			DefaultTeam:    "default-team",
			DefaultChannel: "default-channel",
		}

		api := &plugintest.API{}
		api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
			*dest.(*configuration) = *config
			return nil
		})
		api.On("GetUserByUsername", "username").Return(&model.User{Id: "userId"}, nil)
		api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
		api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(&model.Channel{Id: "channelId"}, nil)
		p.SetAPI(api)

		result := p.OnConfigurationChange()
		if result != nil {
			t.Error("encountered an error within configuration change validation")
		}
	})

	t.Run("load configuration error", func(t *testing.T) {
		p := &Plugin{}
		api := &plugintest.API{}
		api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return((*model.AppError)(nil))
		p.SetAPI(api)

		result := p.OnConfigurationChange()
		if result == nil {
			t.Error("load plugin configuration error did not propagate error")
		}
	})

	t.Run("ensureDefaultUserExists error", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{Username: "username"}

		api := &plugintest.API{}
		api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
			*dest.(*configuration) = *config
			return nil
		})
		api.On("GetUserByUsername", "username").Return(nil, nil)
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		result := p.OnConfigurationChange()
		if result == nil {
			t.Error("ensure default user exists error did not propagate error")
		}
	})

	t.Run("ensureDefaultTeamExists error", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{
			Username:    "username",
			DefaultTeam: "default-team",
		}

		api := &plugintest.API{}
		api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
			*dest.(*configuration) = *config
			return nil
		})
		api.On("GetUserByUsername", "username").Return(&model.User{Id: "userId"}, nil)
		api.On("GetTeamByName", "default-team").Return(nil, nil)
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		result := p.OnConfigurationChange()
		if result == nil {
			t.Error("ensure default team exists error did not propagate error")
		}
	})

	t.Run("ensureDefaultChannelExists error", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{
			Username:       "username",
			DefaultTeam:    "default-team",
			DefaultChannel: "default-channel",
		}

		api := &plugintest.API{}
		api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
			*dest.(*configuration) = *config
			return nil
		})
		api.On("GetUserByUsername", "username").Return(&model.User{Id: "userId"}, nil)
		api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
		api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(nil, nil)
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		result := p.OnConfigurationChange()
		if result == nil {
			t.Error("ensure default channel exists error did not propagate error")
		}
	})
}

func TestEnsureDefaultTeamExists(t *testing.T) {
	t.Run("returns team id", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{DefaultTeam: "default-team"}

		api := &plugintest.API{}
		api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
		p.SetAPI(api)

		result, err := p.ensureDefaultTeamExists(config)
		if result != "teamId" {
			t.Errorf("Expected: %s Actual: %s", "teamId", result)
		}
		if err != nil {
			t.Errorf("Unexpected error returned")
		}
	})

	t.Run("default team config not set", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{}

		result, err := p.ensureDefaultTeamExists(config)
		if result != "" {
			t.Errorf("Expected: %s Actual: %s", "\"\"", result)
		}
		if err != nil {
			t.Error("Empty default team config returned an error")
		}
	})

}

func TestEnsureDefaultChannelExists(t *testing.T) {
	t.Run("returns channel id", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{
			DefaultTeam:    "default-team",
			DefaultChannel: "default-channel",
		}

		api := &plugintest.API{}
		api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(&model.Channel{Id: "channelId"}, nil)
		p.SetAPI(api)

		result, err := p.ensureDefaultChannelExists(config)
		if result != "channelId" {
			t.Errorf("Expected: %s Actual: %s", "teamId", result)
		}
		if err != nil {
			t.Errorf("Unexpected error returned")
		}
	})

	t.Run("default channel config not set", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{}

		result, err := p.ensureDefaultChannelExists(config)
		if result != "" {
			t.Errorf("Expected: %s Actual: %s", "\"\"", result)
		}
		if err != nil {
			t.Error("Empty default channel config returned an error")
		}
	})

	t.Run("default channel config set without default team", func(t *testing.T) {
		p := &Plugin{}
		config := &configuration{DefaultChannel: "default-channel"}

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		result, err := p.ensureDefaultChannelExists(config)
		if result != "" {
			t.Errorf("Expected: %s Actual: %s", "\"\"", result)
		}
		if err == nil {
			t.Error("Default channel without default team did not return error")
		}
	})

}
