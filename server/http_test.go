package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func loadJsonFile(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func TestServeHttp(t *testing.T) {
	emptyBody := []byte("{}")
	itemLink := "https://rollbar.com/item/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
	occurrenceLink := "https://rollbar.com/occurrence/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
	attachmentFields := []*model.SlackAttachmentField{
		&model.SlackAttachmentField{
			Short: true,
			Title: "Environment",
			Value: "live",
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Framework",
			Value: "flask",
		},
		&model.SlackAttachmentField{
			Short: true,
			Title: "Links",
			Value: fmt.Sprintf("[Item](%s) | [Occurrence](%s)", itemLink, occurrenceLink),
		},
	}
	nonNotifyAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    attachmentFields,
			Title:     "New Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}
	withNotifyAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] New Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    attachmentFields,
			Pretext:   "@daniel, @eric",
			Title:     "New Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}
	withOccurrenceNotifyAttachment := []*model.SlackAttachment{
		&model.SlackAttachment{
			Color:     "#ff0000",
			Fallback:  "[live] Occurrence - Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
			Fields:    attachmentFields,
			Pretext:   "@daniel, @eric",
			Title:     "Occurrence - Error",
			TitleLink: itemLink,
			Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
		},
	}

	nonNotifyOverridePost := &model.Post{
		ChannelId: "existingChannelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   nonNotifyAttachment,
		},
	}
	nonNotifyPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   nonNotifyAttachment,
		},
	}
	withNotifyPost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   withNotifyAttachment,
		},
	}
	occurrencePost := &model.Post{
		ChannelId: "channelId",
		UserId:    "userId",
		Type:      model.POST_SLACK_ATTACHMENT,
		Props: map[string]interface{}{
			"from_webhook":  "true",
			"use_user_icon": "true",
			"attachments":   withOccurrenceNotifyAttachment,
		},
	}

	for name, test := range map[string]struct {
		SetupAPI         func(api *plugintest.API) *plugintest.API
		Method           string
		Url              string
		Body             []byte
		Configuration    *configuration
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"error - non-notify defaults to 404": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			Url:              "/",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: "404 page not found\n",
		},
		"error - non-post method not allowed": {
			SetupAPI:         func(api *plugintest.API) *plugintest.API { return api },
			Method:           "GET",
			Url:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{},
			ExpectedStatus:   http.StatusMethodNotAllowed,
			ExpectedResponse: "Method not allowed.\n",
		},
		"error - unauthenticated request": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			Url:              "/notify",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: "Unauthenticated.\n",
		},
		"error - no configured team, no team in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method:           "POST",
			Url:              "/notify?auth=abc123",
			Body:             emptyBody,
			Configuration:    &configuration{Secret: "abc123"},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'team' query parameter.\n",
		},
		"error - no configured channel, no channel in query params": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Missing 'channel' query parameter.\n",
		},
		"error - no configured team, api.get query team not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetTeamByName", "nonexistentQueryTeam").Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&team=nonexistentQueryTeam",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret:    "abc123",
				channelId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Team 'nonexistentQueryTeam' does not exist.\n",
		},
		"error - no configured channel, api.get query channel not found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogWarn", mock.Anything).Return(nil)
				api.On("GetChannelByName", "teamId", "nonexistentQueryChannel", false).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&channel=nonexistentQueryChannel",
			Body:   emptyBody,
			Configuration: &configuration{
				Secret: "abc123",
				teamId: "teamId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "Channel 'nonexistentQueryChannel' does not exist.\n",
		},
		"error - good configs, request body json decode to rollbar err": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogError", mock.Anything).Return(nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   nil, // can't decode nil
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: "EOF\n",
		},
		"error - failed to create post for whatever reason": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), nil)
				api.On("LogError", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: "server/http: error, detailed error\n",
		},
		"ok - error in KVGet for notify users logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(""), &model.AppError{})
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", nonNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - error in json unmarshal for notify users logged and ignored": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"this":error`), nil)
				api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)
				api.On("CreatePost", nonNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - query params team, channel present and exist": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "existingChannelId").Return([]byte(""), nil)
				api.On("GetTeamByName", "existingTeam").Return(&model.Team{Id: "existingTeamId"}, nil)
				api.On("GetChannelByName", "existingTeamId", "existingChannel", false).Return(&model.Channel{Id: "existingChannelId"}, nil)
				api.On("CreatePost", nonNotifyOverridePost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123&team=existingTeam&channel=existingChannel",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - notifies users in pretext": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", withNotifyPost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "new_item.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
		"ok - occurrence json": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("KVGet", "channelId").Return([]byte(`{"daniel":true,"eric":true}`), nil)
				api.On("CreatePost", occurrencePost).Return(nil, nil)
				return api
			},
			Method: "POST",
			Url:    "/notify?auth=abc123",
			Body:   loadJsonFile(t, "every_occurrence.json"),
			Configuration: &configuration{
				Secret:    "abc123",
				userId:    "userId",
				teamId:    "teamId",
				channelId: "channelId",
			},
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := &RollbarPlugin{}
			p.SetAPI(api)
			p.setConfiguration(test.Configuration)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(test.Method, test.Url, bytes.NewReader(test.Body))

			p.ServeHTTP(nil, w, r)

			response := w.Result()
			body, _ := ioutil.ReadAll(response.Body)

			if response.StatusCode != test.ExpectedStatus {
				t.Errorf("Expected status: %d\nActual: %d", test.ExpectedStatus, response.StatusCode)
			}
			if string(body) != test.ExpectedResponse {
				t.Errorf("Expected response: %s\nActual: %s", test.ExpectedResponse, string(body))
			}
		})
	}
}
