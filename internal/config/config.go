package config

import (
	"encoding/json"
	"os"
)

func Load() (Config, error) {
	return LoadFile("config.json")
}

func LoadFile(path string) (Config, error) {
	var conf Config
	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

type Config struct {
	Values []string `json:"values"`
	Port   int      `json:"port"`

	ReFresh        int `json:"refresh"`
	AutoUpdatePush int `json:"autoUpdatePush"`

	ListHeight int    `json:"listHeight"`
	WebTitle   string `json:"webTitle"`
	WebDes     string `json:"webDes"`

	Keywords []string `json:"keywords"`
	Notify   Notify   `json:"notify"`
	Archives string   `json:"archives"`
}

type Notify struct {
	FeiShu   FeiShu   `json:"feishu"`
	Telegram Telegram `json:"telegram"`
	Dingtalk Dingtalk `json:"dingtalk"`
}

type FeiShu struct {
	API string `json:"api"`
}

type Dingtalk struct {
	Webhook string `json:"webhook"`
	Sign    string `json:"sign"`
}

type Telegram struct {
	ChatId string `json:"chat_id"`
	API    string `json:"api"`
	Token  string `json:"token"`
}

func (older Config) GetIncrement(newer Config) []string {
	urlMap := make(map[string]struct{})
	increment := make([]string, 0, len(newer.Values))

	for _, item := range older.Values {
		urlMap[item] = struct{}{}
	}
	for _, item := range newer.Values {
		if _, ok := urlMap[item]; ok {
			continue
		}
		increment = append(increment, item)
	}
	return increment
}
