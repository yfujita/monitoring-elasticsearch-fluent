package slack

import (
	"net/http"
	"encoding/json"
	"bytes"
	"time"
	"io/ioutil"
	"errors"
)

type Bot struct {
	url      string
	channel  string
	botName  string
	faceIcon string
}

func NewBot(url, channel, botName, faceIcon string) *Bot {
	bot := new(Bot)
	bot.url = url
	bot.channel = channel
	bot.botName = botName
	bot.faceIcon = faceIcon
	return bot
}

func (bot *Bot) Message(message, webMessage string) error {
	textMap := make(map[string]string)
	textMap["channel"] = bot.channel
	textMap["username"] = bot.botName
	textMap["icon_emoji"] = bot.faceIcon
	textMap["text"] = message + "\n" + webMessage
	text, _ := json.Marshal(textMap)

	req, err := http.NewRequest(
		"POST",
		bot.url,
		bytes.NewBuffer([]byte(text)),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(15 * time.Second) }
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(b))
	}
	return nil
}


