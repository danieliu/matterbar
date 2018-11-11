package main

import (
	"bytes"
	"text/template"
)

const DefaultTemplate = "[{{ $message := .Data.Item.LastOccurrence.Body.Message.Body }}{{ if $message }}{{ $message }}{{ else }}{{.Data.Item.LastOccurrence.Body.Trace.Exception.Message}}{{ end }}](https://rollbar.com/<org>/<project>/items/{{ .Data.Item.Counter }})\n" +
	"{{ .EventName }} - [{{ .Data.Item.Environment }}] - {{ .Data.Item.LastOccurrence.Level }}"

type Rollbar struct {
	EventName string `json:"event_name"`
	Data      struct {
		Item struct {
			ActivatingOccurrenceId   int      `json:"activating_occurrence_id"`
			AssignedUserId           *int     `json:"assigned_user_id"`
			Counter                  int      `json:"counter"`
			Environment              string   `json:"environment"`
			FirstOccurrenceId        int      `json:"first_occurrence_id"`
			FirstOccurrenceTimestamp int      `json:"first_occurrence_timestamp"`
			Framework                int      `json:"framework"`
			Hash                     string   `json:"hash"`
			Id                       int      `json:"id"`
			IntegrationsData         struct{} `json:"integrations_data"`
			LastActivatedTimestamp   int      `json:"last_activated_timestamp"`
			LastModifiedBy           *int     `json:"last_modified_by"`
			LastOccurrence           struct {
				Body struct {
					Message struct {
						Body string `json:"body"`
					} `json:"message"`
					Trace struct {
						Exception struct {
							Class   string `json:"class"`
							Message string `json:"message"`
						} `json:"exception"`
						Frames []struct {
							Code     string `json:"code"`
							Filename string `json:"filename"`
							LineNo   int    `json:"lineno"`
							Locals   struct {
								Builtins string `json:"__builtins__"`
								Doc      string `json:"__doc__"`
								File     string `json:"__file__"`
								Name     string `json:"__name__"`
								Package  string `json:"__package__"`
								Rollbar  string `json:"rollbar"`
							} `json:"locals"`
							Method string `json:"method"`
						} `json:"frames"`
					} `json:"trace"`
				} ` json:"body"`
				Environment string `json:"environment"`
				Framework   string `json:"framework"`
				Language    string `json:"language"`
				Level       string `json:"level"`
				Metadata    struct {
					AccessToken       string `json:"access_token"`
					ApiServerHostname string `json:"api_server_hostname"`
					CustomerTimestamp int    `json:"customer_timestamp"`
					Debug             struct {
						Routes struct {
							Counters struct {
								PostItem int `json:"post_item"`
							} `json:"counters"`
							StartTime int `json:"start_time"`
						} `json:"routes"`
					} `json:"debug"`
					TimestampMs int `json:"timestamp_ms"`
				} `json:"metadata"`
				Notifier struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"notifier"`
				Server struct {
					Argv []string `json:"argv"`
					Host string   `json:"host"`
					Pid  int      `json:"pid"`
				} `json:"server"`
				Timestamp int    `json:"timestamp"`
				Uuid      string `json:"uuid"`
			} `json:"last_occurrence"`
			LastOccurrenceId        int     `json:"last_occurrence_id"`
			LastOccurrenceTimestamp int     `json:"last_occurrence_timestamp"`
			Level                   int     `json:"level"`
			LevelLock               int     `json:"level_lock"`
			Platform                int     `json:"platform"`
			ProjectId               int     `json:"project_id"`
			PublicItemId            *int    `json:"public_item_id"`
			ResolvedInVersion       *string `json:"resolved_in_version"`
			Status                  int     `json:"status"`
			Title                   string  `json:"title"`
			TitleLock               int     `json:"title_lock"`
			TotalOccurrences        int     `json:"total_occurrences"`
			UniqueOccurrences       *int    `json:"unique_occurrences"`
		} `json:"item"`
		Occurrences int `json:"occurrences"`
		Trigger     struct {
			Threshold             int    `json:"threshold"`
			WindowSize            int    `json:"window_size"`
			WindowSizeDescription string `json:"window_size_description"`
		} `json:"trigger"`
	} `json:"data"`
}

func (rollbar *Rollbar) interpolateMessage(message string) (string, error) {
	tmpl, err := template.New("post").Parse(message)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, rollbar); err != nil {
		return "", err
	}

	return result.String(), nil
}
