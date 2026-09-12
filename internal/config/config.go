package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPath        = "config.json"
	defaultTelegramAPI = "https://api.telegram.org/bot${token}/sendMessage"
)

func Path() string {
	if path := strings.TrimSpace(os.Getenv("RSS_READER_CONFIG")); path != "" {
		return path
	}
	return DefaultPath
}

func Load() (Config, error) {
	return LoadFile(Path())
}

func LoadFile(path string) (Config, error) {
	var conf Config
	data, err := os.ReadFile(path)
	if err != nil {
		return conf, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return conf, fmt.Errorf("parse config %q: %w", path, err)
	}

	applyDefaults(&conf)
	if err := applyEnvironment(&conf); err != nil {
		return Config{}, err
	}
	if err := conf.Validate(); err != nil {
		return Config{}, err
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

func (conf Config) Validate() error {
	if conf.Port < 1 || conf.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if conf.ReFresh <= 0 {
		return fmt.Errorf("refresh must be greater than 0 minutes")
	}
	if conf.AutoUpdatePush < 0 {
		return fmt.Errorf("autoUpdatePush cannot be negative")
	}
	if conf.ListHeight <= 0 {
		return fmt.Errorf("listHeight must be greater than 0")
	}
	if strings.TrimSpace(conf.Archives) == "" {
		return fmt.Errorf("archives path cannot be empty")
	}
	if (conf.Notify.Telegram.Token == "") != (conf.Notify.Telegram.ChatId == "") {
		return fmt.Errorf("telegram token and chat_id must be configured together")
	}
	return nil
}

func applyDefaults(conf *Config) {
	if conf.Port == 0 {
		conf.Port = 8080
	}
	if conf.ReFresh == 0 {
		conf.ReFresh = 5
	}
	if conf.ListHeight == 0 {
		conf.ListHeight = 600
	}
	if strings.TrimSpace(conf.Archives) == "" {
		conf.Archives = "archives.txt"
	}
	if strings.TrimSpace(conf.Notify.Telegram.API) == "" {
		conf.Notify.Telegram.API = defaultTelegramAPI
	}
}

func applyEnvironment(conf *Config) error {
	if value := strings.TrimSpace(os.Getenv("RSS_READER_PORT")); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("RSS_READER_PORT must be an integer: %w", err)
		}
		conf.Port = port
	}
	if value := strings.TrimSpace(os.Getenv("RSS_READER_ARCHIVES")); value != "" {
		conf.Archives = value
	}
	if value := os.Getenv("RSS_READER_FEISHU_API"); value != "" {
		conf.Notify.FeiShu.API = value
	}
	if value := os.Getenv("RSS_READER_DINGTALK_WEBHOOK"); value != "" {
		conf.Notify.Dingtalk.Webhook = value
	}
	if value := os.Getenv("RSS_READER_DINGTALK_SIGN"); value != "" {
		conf.Notify.Dingtalk.Sign = value
	}
	if value := os.Getenv("RSS_READER_TELEGRAM_API"); value != "" {
		conf.Notify.Telegram.API = value
	}
	if value := os.Getenv("RSS_READER_TELEGRAM_CHAT_ID"); value != "" {
		conf.Notify.Telegram.ChatId = value
	}
	if value := os.Getenv("RSS_READER_TELEGRAM_TOKEN"); value != "" {
		conf.Notify.Telegram.Token = value
	}
	return nil
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
