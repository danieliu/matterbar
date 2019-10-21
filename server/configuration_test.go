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
			Secret:         "secret",
			teamId:         "teamId",
			channelId:      "channelId",
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
		plugin := &RollbarPlugin{}
		if configuration := plugin.getConfiguration(); configuration == nil {
			t.Error("Expected configuration to not be nil")
		}
	})

	t.Run("changing configuration", func(t *testing.T) {
		plugin := &RollbarPlugin{}
		configuration1 := &configuration{teamId: "123"}
		plugin.setConfiguration(configuration1)
		if configuration1 != plugin.getConfiguration() {
			t.Errorf("Configurations not equal. %s != %s",
				configuration1,
				plugin.getConfiguration())
		}

		configuration2 := &configuration{teamId: "456"}
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
		plugin := &RollbarPlugin{}
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
		plugin := &RollbarPlugin{}

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
	for name, test := range map[string]struct {
		SetupAPI              func(*plugintest.API, *configuration) *plugintest.API
		Configuration         *configuration
		ExpectedConfiguration *configuration
		ShouldError           bool
	}{
		"ok": {
			SetupAPI: func(api *plugintest.API, config *configuration) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
					*dest.(*configuration) = *config
					return nil
				})
				api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
				api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(&model.Channel{Id: "channelId"}, nil)
				api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).Return(nil)
				return api
			},
			Configuration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
			},
			ExpectedConfiguration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
				teamId:         "teamId",
				channelId:      "channelId",
			},
			ShouldError: false,
		},
		"load configuration error": {
			SetupAPI: func(api *plugintest.API, config *configuration) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(&model.AppError{})
				return api
			},
			Configuration:         &configuration{},
			ExpectedConfiguration: &configuration{},
			ShouldError:           true,
		},
		"ensureDefaultTeamExists error": {
			SetupAPI: func(api *plugintest.API, config *configuration) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
					*dest.(*configuration) = *config
					return nil
				})
				api.On("GetTeamByName", "default-team").Return(nil, nil)
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Configuration: &configuration{
				DefaultTeam: "default-team",
			},
			ExpectedConfiguration: &configuration{
				DefaultTeam: "default-team",
			},
			ShouldError: true,
		},
		"ensureDefaultChannelExists error": {
			SetupAPI: func(api *plugintest.API, config *configuration) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
					*dest.(*configuration) = *config
					return nil
				})
				api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
				api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(nil, nil)
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Configuration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
			},
			ExpectedConfiguration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
			},
			ShouldError: true,
		},
		"register command error": {
			SetupAPI: func(api *plugintest.API, config *configuration) *plugintest.API {
				api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Return(func(dest interface{}) error {
					*dest.(*configuration) = *config
					return nil
				})
				api.On("GetTeamByName", "default-team").Return(&model.Team{Id: "teamId"}, nil)
				api.On("GetChannelByNameForTeamName", "default-team", "default-channel", false).Return(&model.Channel{Id: "channelId"}, nil)
				api.On("RegisterCommand", mock.AnythingOfType("*model.Command")).Return(&model.AppError{Message: "error"})
				return api
			},
			Configuration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
			},
			ExpectedConfiguration: &configuration{
				DefaultTeam:    "default-team",
				DefaultChannel: "default-channel",
			},
			ShouldError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{}, test.Configuration)
			defer api.AssertExpectations(t)
			p := &RollbarPlugin{}
			p.SetAPI(api)
			p.setConfiguration(test.Configuration)

			err := p.OnConfigurationChange()
			if *test.ExpectedConfiguration != *p.getConfiguration() {
				t.Errorf("Expected: %s\nActual: %s", *test.ExpectedConfiguration, *p.getConfiguration())
			}
			if test.ShouldError {
				if err == nil {
					t.Error("Expected an error, got nil instead")
				}
			} else {
				if err != nil {
					t.Errorf("Error: %s", err.Error())
				}
			}
		})
	}
}

func TestEnsureDefaultTeamExists(t *testing.T) {
	t.Run("returns team id", func(t *testing.T) {
		p := &RollbarPlugin{}
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
		p := &RollbarPlugin{}
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
		p := &RollbarPlugin{}
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
		p := &RollbarPlugin{}
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
		p := &RollbarPlugin{}
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
