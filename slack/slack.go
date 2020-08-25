package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const failColor = "#CC0000"
const okColor = "#00CC00"

type Alerter interface {
	Alert(title string, subAlerts []SubAlert, ok bool)
}

type slackAlerter struct {
	webHook string
}

type SubAlert struct {
	Title  string
	Output string
}

func NewSlackAlerter(webHook string) Alerter {
	return &slackAlerter{
		webHook: webHook,
	}
}

func escapeChars(s string) string {
	s = strings.Replace(s, "&", "&amp;", -1)
	s = strings.Replace(s, "<", "&lt;", -1)
	s = strings.Replace(s, ">", "&gt;", -1)
	return s
}

func (s *slackAlerter) Alert(title string, subAlerts []SubAlert, ok bool) {
	var fallback string
	var color string
	if ok {
		fallback = fmt.Sprintf("ok")
		color = okColor
	} else {
		fallback = fmt.Sprintf("failure")
		color = failColor
	}

	fields := []SlackField{}
	for _, alert := range subAlerts {
		fields = append(fields, SlackField{
			Title: alert.Title,
			Value: alert.Output,
			Short: false,
		})
	}

	notification := &SlackNotification{
		Attachments: []SlackAttachment{
			{
				Fallback:   fallback,
				Color:      color,
				Title:      title,
				TitleLink:  "https://formulae.brew.sh/formula/earthly",
				MarkDownIn: []string{"pretext", "text"},
				Fields:     fields,
			},
		},
	}

	err := sendNotification(s.webHook, notification)
	if err != nil {
		panic(err)
	}
}

func sendNotification(webhook string, notification *SlackNotification) error {
	buffer := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buffer).Encode(notification); err != nil {
		return err
	}

	res, err := http.Post(webhook, "application/json", buffer)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		buffer.Reset()
		if _, err := io.Copy(buffer, res.Body); err != nil {
			return err
		}

		return &SlackError{
			Code: res.StatusCode,
			Body: buffer.String(),
		}
	}
	return nil
}

type SlackError struct {
	Code int
	Body string
}

func (e SlackError) Error() string {
	return fmt.Sprintf("slack webhook returned %d: %s", e.Code, e.Body)
}

type SlackNotification struct {
	Text        string            `json:"text"`
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
	Fallback   string       `json:"fallback"`
	Pretext    string       `json:"pretext,omitempty"`
	Text       string       `json:"text,omitempty"`
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitmepty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Fields     []SlackField `json:"fields"`
	MarkDownIn []string     `json:"mrkdwn_in"`
	Footer     string       `json:"footer"`
}

type SlackField struct {
	Title string      `json:"title"`
	Value interface{} `json:"value"`
	Short bool        `json:"short,omitempty"`
}
