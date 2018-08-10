/*
Copyright 2016 Skippbox, Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mattermost

import (
	"fmt"
	"log"
	"os"

	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bitnami-labs/kubewatch/config"
	kbEvent "github.com/bitnami-labs/kubewatch/pkg/event"
)

var mattermostColors = map[string]string{
	"Normal":  "#00FF00",
	"Warning": "#FFFF00",
	"Danger":  "#FF0000",
}

var mattermostErrMsg = `
%s

You need to set Mattermost url, channel and username for Mattermost notify,
using "--channel/-c", "--url/-u" and "--username/-n", or using environment variables:

export KW_MATTERMOST_CHANNELID=mattermost_channel
export KW_MATTERMOST_URL=mattermost_url
export KW_MATTERMOST_USERNAME=mattermost_username

Command line flags will override environment variables

`

// Mattermost handler implements handler.Handler interface,
// Notify event to Mattermost channel
//type Mattermost struct {
//	Channel  string
//	Url      string
//	Username string
//}

type Mattermost struct {
	ChannelId  string
	Url      string
	Username string
	Token string
}

type MattermostMessageV4 struct {
	ChannelId string `json:"channel_id"`
	Message string `json:"message"`
}

//type MattermostMessage struct {
//	Channel      string                         `json:"channel"`
//	Username     string                         `json:"username"`
//	IconUrl      string                         `json:"icon_url"`
//	Text         string                         `json:"text"`
//	Attachements []MattermostMessageAttachement `json:"attachments"`
//}

//type MattermostMessageAttachement struct {
//	Title string `json:"title"`
//	Color string `json:"color"`
//}

// Init prepares Mattermost configuration
func (m *Mattermost) Init(c *config.Config) error {
	channelId := c.Handler.Mattermost.ChannelId
	url := c.Handler.Mattermost.Url
	token := c.Handler.Mattermost.Token

	if channelId == "" {
		channelId = os.Getenv("KW_MATTERMOST_CHANNELID")
	}

	if url == "" {
		url = os.Getenv("KW_MATTERMOST_URL")
	}

	if token == "" {
		token = os.Getenv("KW_MATTERMOST_TOKEN")
	}

	m.ChannelId = channelId
	m.Url = url
	m.Token = token

	return checkMissingMattermostVars(m)
}

func (m *Mattermost) ObjectCreated(obj interface{}) {
	notifyMattermost(m, obj, "created")
}

func (m *Mattermost) ObjectDeleted(obj interface{}) {
	notifyMattermost(m, obj, "deleted")
}

func (m *Mattermost) ObjectUpdated(oldObj, newObj interface{}) {
	notifyMattermost(m, newObj, "updated")
}

func notifyMattermost(m *Mattermost, obj interface{}, action string) {
	e := kbEvent.New(obj, action)

	mattermostMessage := prepareMattermostMessage(e, m)

	err := postMessage(m.Url, m.Token, mattermostMessage)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to channel %s at %s", m.ChannelId, time.Now())
}

func checkMissingMattermostVars(s *Mattermost) error {
	if s.ChannelId == "" || s.Url == "" || s.Token == "" {
		return fmt.Errorf(mattermostErrMsg, "Missing Mattermost channelId, url or token")
	}

	return nil
}

func prepareMattermostMessage(e kbEvent.Event, m *Mattermost) *MattermostMessage {

	return &MattermostMessageV4{
		ChannelId:  m.ChannelId,
		Message: e.Message(), // TODO markdown message
	}
}

func postMessage(url, token string, mattermostMessage *MattermostMessage) error {
	message, err := json.Marshal(mattermostMessage)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url + "/api/v4/posts", bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer " + token)

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}
