package main

import (
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func TestOnActivate(t *testing.T) {
	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botID := "bot-id"
	path, err := filepath.Abs("..")
	if err != nil {
		t.Errorf("absolute path error: %s", err.Error())
	}

	for name, test := range map[string]struct {
		SetupAPI     func(*plugintest.API) *plugintest.API
		SetupHelpers func(*plugintest.Helpers) *plugintest.Helpers
		ShouldError  bool
	}{
		"EnsureBot fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API { return api },
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return("", &model.AppError{})
				return helpers
			},
			ShouldError: true,
		},
		"GetBundlePath fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("", &model.AppError{})
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return(botID, nil)
				return helpers
			},
			ShouldError: true,
		},
		"profileImage nonexistent": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return("/tmp", nil)
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return(botID, nil)
				return helpers
			},
			ShouldError: true,
		},
		"SetProfileImage fails": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return(path, nil)
				api.On("SetProfileImage", botID, mock.Anything).Return(&model.AppError{})
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return(botID, nil)
				return helpers
			},
			ShouldError: true,
		},
		"ok bot id set": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("GetBundlePath").Return(path, nil)
				api.On("SetProfileImage", botID, mock.Anything).Return(nil)
				return api
			},
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return(botID, nil)
				return helpers
			},
			ShouldError: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)

			helpers := &plugintest.Helpers{}
			if test.SetupHelpers != nil {
				helpers = test.SetupHelpers(helpers)
				defer helpers.AssertExpectations(t)
			}

			p := &RollbarPlugin{}
			p.SetAPI(api)
			p.SetHelpers(helpers)
			p.setConfiguration(&configuration{})

			err := p.OnActivate()
			if test.ShouldError {
				if err == nil {
					t.Error("Expected an error, got nil instead")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err.Error())
				}
				if botID != p.botUserID {
					t.Errorf("Expected botID: %s\nActual: %s", botID, p.botUserID)
				}
			}
		})
	}
}
