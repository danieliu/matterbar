package main

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

type GetUsernameTest struct {
	Map      map[string]bool
	Expected string
}

func TestGetUsernameList(t *testing.T) {
	noUser := make(map[string]bool)
	singleUser := map[string]bool{"daniel": true}
	multipleUser := map[string]bool{"daniel": true, "eric": true}

	testcases := []GetUsernameTest{
		{noUser, "None"},
		{singleUser, "@daniel"},
		{multipleUser, "@daniel, @eric"},
	}

	for _, test := range testcases {
		result := GetUsernameList(test.Map)
		if result != test.Expected {
			t.Errorf("Expected: %s\nActual: %s", test.Expected, result)
		}
	}
}

func TestExecuteCommand(t *testing.T) {
	for name, test := range map[string]struct {
		SetupAPI     func(*plugintest.API) *plugintest.API
		Command      string
		ExpectedText string
	}{
		"No argument error": {
			SetupAPI:     func(api *plugintest.API) *plugintest.API { return api },
			Command:      "/rollbar",
			ExpectedText: usageErrorMessage,
		},
		"too many tokens/arguments": {
			SetupAPI:     func(api *plugintest.API) *plugintest.API { return api },
			Command:      "/rollbar notify @daniel @eric",
			ExpectedText: usageErrorMessage,
		},
		"KVGet error": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return(nil, &model.AppError{})
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				return api
			},
			Command:      "/rollbar list",
			ExpectedText: genericErrorMessage,
		},
		"json.Unmarshal error": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte("error"), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				return api
			},
			Command:      "/rollbar list",
			ExpectedText: genericErrorMessage,
		},
		"list command, extra arguments": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel": true}`), nil)
				return api
			},
			Command:      "/rollbar list @daniel",
			ExpectedText: "Usage: `/rollbar list`",
		},
		"list command, ok": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel": true}`), nil)
				return api
			},
			Command:      "/rollbar list",
			ExpectedText: fmt.Sprintf(usersListMessage, "@daniel"),
		},
		"notify command, not enough arguments": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel": true}`), nil)
				return api
			},
			Command:      "/rollbar notify",
			ExpectedText: "Usage: `/rollbar notify @username`",
		},
		"notify command, user does not exist": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel": true}`), nil)
				api.On("GetUserByUsername", "daniel").Return(nil, nil)
				return api
			},
			Command:      "/rollbar notify @daniel",
			ExpectedText: "User `daniel` not found.",
		},
		"notify command, user already being notified": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel": true}`), nil)
				api.On("GetUserByUsername", "daniel").Return(&model.User{}, nil)
				return api
			},
			Command:      "/rollbar notify @daniel",
			ExpectedText: "User `daniel` is already being notified.",
		},
		"notify command, adds user to list": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(``), nil)
				api.On("GetUserByUsername", "daniel").Return(&model.User{}, nil)
				api.On("KVSet", "channelId", []byte(`{"daniel":true}`)).Return(nil)
				return api
			},
			Command:      "/rollbar notify @daniel",
			ExpectedText: fmt.Sprintf(usersListMessage, "@daniel"),
		},
		"KVSet error": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(``), nil)
				api.On("GetUserByUsername", "daniel").Return(&model.User{}, nil)
				api.On("KVSet", "channelId", []byte(`{"daniel":true}`)).Return(&model.AppError{})
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				return api
			},
			Command:      "/rollbar notify @daniel",
			ExpectedText: genericErrorMessage,
		},
		"remove command, user already not being notified": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(``), nil)
				api.On("GetUserByUsername", "daniel").Return(&model.User{}, nil)
				return api
			},
			Command:      "/rollbar remove @daniel",
			ExpectedText: "User `daniel` is already not being notified.",
		},
		"remove command, removes user from list": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true}`), nil)
				api.On("GetUserByUsername", "daniel").Return(&model.User{}, nil)
				api.On("KVSet", "channelId", []byte(`{}`)).Return(nil)
				return api
			},
			Command:      "/rollbar remove @daniel",
			ExpectedText: fmt.Sprintf(usersListMessage, "None"),
		},
		"not a supported command": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true}`), nil)
				return api
			},
			Command:      "/rollbar show @daniel",
			ExpectedText: usageErrorMessage,
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := &RollbarPlugin{}
			p.SetAPI(api)

			r, err := p.ExecuteCommand(nil, &model.CommandArgs{
				Command:   test.Command,
				ChannelId: "channelId",
			})

			if err != nil {
				t.Errorf("Error: %s", err.Error())
			}

			if r.ResponseType != model.COMMAND_RESPONSE_TYPE_EPHEMERAL {
				t.Errorf("Wrong ResponseType: %s", r.ResponseType)
			}

			if r.Text != test.ExpectedText {
				t.Errorf("Expected: %s\nActual: %s", test.ExpectedText, r.Text)
			}
		})
	}
}
