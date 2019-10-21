package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
)

func TestOnActivate(t *testing.T) {
	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botID := "bot-id"

	for name, test := range map[string]struct {
		SetupHelpers func(*plugintest.Helpers) *plugintest.Helpers
		ShouldError  bool
	}{
		"EnsureBot fails": {
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return("", &model.AppError{})
				return helpers
			},
			ShouldError: true,
		},
		"ok bot id set": {
			SetupHelpers: func(helpers *plugintest.Helpers) *plugintest.Helpers {
				helpers.On("EnsureBot", bot).Return(botID, nil)
				return helpers
			},
			ShouldError: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			helpers := &plugintest.Helpers{}
			if test.SetupHelpers != nil {
				helpers = test.SetupHelpers(helpers)
				defer helpers.AssertExpectations(t)
			}

			p := &RollbarPlugin{}
			p.SetHelpers(helpers)
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
