package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
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

type ServeHTTPTest struct {
	name string

	method string
	url    string
	body   map[string]interface{}

	expectedStatus   int
	expectedResponse string
}

func TestServeHttp(t *testing.T) {
	emptyBody := make(map[string]interface{})
	testcases := []ServeHTTPTest{
		{"error - non-notify defaults 404", "GET", "/", emptyBody, http.StatusNotFound, "404 page not found\n"},
		{"error - non-post method not allowed", "GET", "/notify", emptyBody, http.StatusMethodNotAllowed, "Method not allowed.\n"},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			p := &RollbarPlugin{}
			w := httptest.NewRecorder()

			payload, _ := json.Marshal(test.body)
			r := httptest.NewRequest(test.method, test.url, bytes.NewReader(payload))

			p.ServeHTTP(&plugin.Context{}, w, r)

			response := w.Result()
			body, _ := ioutil.ReadAll(response.Body)

			if response.StatusCode != test.expectedStatus {
				t.Errorf("Expected status: %d Actual: %d", test.expectedStatus, response.StatusCode)
			}
			if string(body) != test.expectedResponse {
				t.Errorf("Did not expect response body:%s", string(body))
			}
		})
	}

	// TODO: use interfaces to have a custom setup as part of the ServeHTTPTest
	// struct and add the following tests to testcases
	t.Run("error - unauthenticated request", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify", nil)

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status: %d Actual: %d", http.StatusUnauthorized, response.StatusCode)
		}
		if string(body) != "Unauthenticated.\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - no team configuration, no team in query", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify?auth=abc123", nil)

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "Missing 'team' query parameter.\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - no channel config, no channel in query", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify?auth=abc123", nil)

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", teamId: "teamId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "Missing 'channel' query parameter.\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - no team config, api.get query team not found", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify?auth=abc123&team=nonexistentQueryTeam", nil)

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		api.On("GetTeamByName", "nonexistentQueryTeam").Return(nil, nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", channelId: "channelId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "Team 'nonexistentQueryTeam' does not exist.\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - no channel config, api.get query channel not found", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify?auth=abc123&channel=nonexistentQueryChannel", nil)

		api := &plugintest.API{}
		api.On("LogWarn", mock.Anything).Return(nil)
		api.On("GetChannelByName", "teamId", "nonexistentQueryChannel", false).Return(nil, nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", teamId: "teamId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "Channel 'nonexistentQueryChannel' does not exist.\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - good configs, request body json decode to rollbar err", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/notify?auth=abc123", nil) // can't decode nil

		api := &plugintest.API{}
		api.On("LogError", mock.Anything).Return(nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", teamId: "teamId", channelId: "channelId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "EOF\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("error - failed to create post for whatever reason", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		payload, _ := json.Marshal(make(map[string]interface{}))
		r := httptest.NewRequest("POST", "/notify?auth=abc123", bytes.NewReader(payload))

		api := &plugintest.API{}
		api.On("LogError", mock.Anything).Return(nil)
		api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, &model.AppError{Where: "server/http", Message: "error", DetailedError: "detailed error"})
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", teamId: "teamId", channelId: "channelId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "server/http: error, detailed error\n" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})

	t.Run("ok - query team, channel present and exist", func(t *testing.T) {
		p := &RollbarPlugin{}
		w := httptest.NewRecorder()
		payload := loadJsonFile(t, "new_item.json")
		r := httptest.NewRequest("POST", "/notify?auth=abc123&team=existingTeam&channel=existingChannel", bytes.NewReader(payload))

		itemLink := "https://rollbar.com/item/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
		occurrenceLink := "https://rollbar.com/occurrence/uuid/?uuid=2e7cbf0a-a3af-402a-ab4f-95e07e5982f8"
		createPostCalledWith := &model.Post{
			ChannelId: "existingChannelId",
			UserId:    "userId",
			Type:      model.POST_SLACK_ATTACHMENT,
			Props: map[string]interface{}{
				"from_webhook":  "true",
				"use_user_icon": "true",
				"attachments": []*model.SlackAttachment{
					&model.SlackAttachment{
						Color:    "#ff0000",
						Fallback: "[live] New Error - TypeError: 'NoneType' object has no attribute '__getitem__'",
						Fields: []*model.SlackAttachmentField{
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
						},
						Title:     "New Error",
						TitleLink: itemLink,
						Text:      "```\nTypeError: 'NoneType' object has no attribute '__getitem__'\n```",
					},
				},
			},
		}

		api := &plugintest.API{}
		api.On("LogError", mock.Anything).Return(nil)
		api.On("GetTeamByName", "existingTeam").Return(&model.Team{Id: "existingTeamId"}, nil)
		api.On("GetChannelByName", "existingTeamId", "existingChannel", false).Return(&model.Channel{Id: "existingChannelId"}, nil)
		api.On("CreatePost", createPostCalledWith).Return(nil, nil)
		p.SetAPI(api)

		config := &configuration{Secret: "abc123", userId: "userId", teamId: "teamId", channelId: "channelId"}
		p.setConfiguration(config)

		p.ServeHTTP(&plugin.Context{}, w, r)

		response := w.Result()
		body, _ := ioutil.ReadAll(response.Body)

		if response.StatusCode != http.StatusOK {
			t.Errorf("Expected status: %d Actual: %d", http.StatusBadRequest, response.StatusCode)
		}
		if string(body) != "" {
			t.Errorf("Did not expect response body: %s", string(body))
		}
	})
}
