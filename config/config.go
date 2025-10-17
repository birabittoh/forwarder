package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/zelenin/go-tdlib/client"
)

type Config struct {
	ApiID                int32
	ApiHash              string
	PhoneNumber          string
	SourceChannelID      int64
	TargetChannelID      int64
	DiscussionGroupID    int64
	CommentTemplate      *client.FormattedText
	CommentNotifications bool
	ShowForwarded        bool
	IgnoreRegex          *regexp.Regexp
	DatabaseDirectory    string
	FilesDirectory       string
	VerbosityLevel       int
}

func New() (*Config, error) {
	cfg := &Config{}

	apiID, err := getEnvInt("API_ID", "")
	if err != nil {
		return nil, fmt.Errorf("invalid API_ID: %w", err)
	}
	cfg.ApiID = int32(apiID)

	cfg.ApiHash = getEnv("API_HASH", "")
	cfg.PhoneNumber = getEnv("PHONE_NUMBER", "")

	cfg.SourceChannelID, err = getEnvInt64("SOURCE_CHANNEL_ID", "")
	if err != nil {
		return nil, fmt.Errorf("invalid SOURCE_CHANNEL_ID: %w", err)
	}

	cfg.TargetChannelID, err = getEnvInt64("TARGET_CHANNEL_ID", "")
	if err != nil {
		return nil, fmt.Errorf("invalid TARGET_CHANNEL_ID: %w", err)
	}

	cfg.DiscussionGroupID, _ = getEnvInt64("DISCUSSION_GROUP_ID", "")

	commentFile := getEnv("COMMENT_TEMPLATE_FILE", "comment.md")
	if commentFile != "" {
		commentBytes, err := os.ReadFile(commentFile)
		if err == nil {
			cfg.CommentTemplate, err = client.ParseMarkdown(&client.ParseMarkdownRequest{
				Text: &client.FormattedText{
					Text: string(commentBytes),
				},
			})

			if err != nil {
				cfg.CommentTemplate = nil
			}
		}
	}

	cfg.CommentNotifications = getEnvBool("ENABLE_COMMENT_NOTIFICATIONS", false)
	cfg.ShowForwarded = getEnvBool("SHOW_FORWARDED", false)

	ignoreRegexStr := getEnv("IGNORE_REGEX", "")
	if ignoreRegexStr != "" {
		cfg.IgnoreRegex, err = regexp.Compile(ignoreRegexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid IGNORE_REGEX: %w", err)
		}
	}

	cfg.DatabaseDirectory = getEnv("DATABASE_DIRECTORY", "./tdlib-db")
	cfg.FilesDirectory = getEnv("FILES_DIRECTORY", "./tdlib-files")

	cfg.VerbosityLevel, err = getEnvInt("VERBOSITY_LEVEL", "2")
	if err != nil {
		return nil, fmt.Errorf("invalid VERBOSITY_LEVEL: %w", err)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key, defaultValue string) (int, error) {
	valStr := getEnv(key, defaultValue)
	return strconv.Atoi(valStr)
}

func getEnvInt64(key, defaultValue string) (int64, error) {
	valStr := getEnv(key, defaultValue)
	return strconv.ParseInt(valStr, 10, 64)
}

func getEnvBool(key string, defaultValue bool) bool {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultValue
	}
	return b
}
