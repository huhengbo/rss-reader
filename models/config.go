package models

import (
	"encoding/json"
	"os"
)

func ParseConf() (Config, error) {
	return ParseConfFile("config.json")
}

func ParseConfFile(path string) (Config, error) {
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

	Keywords []string `json:"keywords"` // 关键词
	Notify   Notify   `json:"notify"`   // 通知方式
	Archives string   `json:"archives"` // 通知方式
}

// Notify 通知方式
type Notify struct {
	FeiShu   FeiShu   `json:"feishu"`
	Telegram Telegram `json:"telegram"`
	Dingtalk Dingtalk `json:"dingtalk"`
}

// FeiShu 飞书
type FeiShu struct {
	//Text string `json:"text"`
	API string `json:"api"`
}

// Dingtalk 钉钉通知配置。
type Dingtalk struct {
	//Text string `json:"text"`
	Webhook string `json:"webhook"`
	Sign    string `json:"sign"`
}

// Telegram 电报通知配置。
type Telegram struct {
	ChatId string `json:"chat_id"`
	//Text   string `json:"text"`
	API   string `json:"api"`
	Token string `json:"token"`
}

func (older Config) GetIncrement(newer Config) []string {
	var (
		urlMap    = make(map[string]struct{})
		increment = make([]string, 0, len(newer.Values))
	)
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
