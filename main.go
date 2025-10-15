package main

import (
	"log"
	"os"
	"regexp"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type Config struct {
	BotToken          string
	SourceChannelID   int64
	DestChannelID     int64
	DiscussionGroupID int64
	IgnoreRegex       *regexp.Regexp
	CommentTemplate   string
}

func loadConfig() (*Config, error) {
	// Carica il file .env
	godotenv.Load()

	// Leggi le variabili di ambiente
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	sourceChannelID, err := strconv.ParseInt(os.Getenv("SOURCE_CHANNEL_ID"), 10, 64)
	if err != nil {
		log.Fatal("SOURCE_CHANNEL_ID must be a valid integer")
	}

	destChannelID, err := strconv.ParseInt(os.Getenv("DEST_CHANNEL_ID"), 10, 64)
	if err != nil {
		log.Fatal("DEST_CHANNEL_ID must be a valid integer")
	}

	discussionGroupID, err := strconv.ParseInt(os.Getenv("DISCUSSION_GROUP_ID"), 10, 64)
	if err != nil {
		log.Fatal("DISCUSSION_GROUP_ID must be a valid integer")
	}

	// Regex per ignorare messaggi (default: #aff)
	regexPattern := os.Getenv("IGNORE_REGEX")
	if regexPattern == "" {
		regexPattern = "#aff|VERISURE|KINDLE"
	}
	ignoreRegex, err := regexp.Compile(regexPattern)
	if err != nil {
		log.Fatalf("Invalid IGNORE_REGEX pattern: %v", err)
	}

	// Leggi il template del commento da file .md
	commentFile := os.Getenv("COMMENT_FILE")
	if commentFile == "" {
		commentFile = "comment.md"
	}

	commentTemplate, err := os.ReadFile(commentFile)
	if err != nil {
		log.Printf("Error reading comment file %s: %v", commentFile, err)
	}

	return &Config{
		BotToken:          botToken,
		SourceChannelID:   sourceChannelID,
		DestChannelID:     destChannelID,
		DiscussionGroupID: discussionGroupID,
		IgnoreRegex:       ignoreRegex,
		CommentTemplate:   string(commentTemplate),
	}, nil
}

func main() {
	// Carica la configurazione
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Crea il bot
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Bot authorized on account %s", bot.Self.UserName)

	// Configura gli aggiornamenti
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Loop principale
	for update := range updates {
		if update.ChannelPost == nil {
			continue
		}

		message := update.ChannelPost

		// Verifica se il messaggio proviene dal canale sorgente
		if message.Chat.ID != config.SourceChannelID {
			continue
		}

		// Ottieni il testo del messaggio
		messageText := message.Text
		if messageText == "" && message.Caption != "" {
			messageText = message.Caption
		}

		// Controlla se il messaggio matcha la regex da ignorare
		if config.IgnoreRegex.MatchString(messageText) {
			log.Printf("Message ignored (matched regex): %s", messageText)
			continue
		}

		// Inoltra il messaggio al canale di destinazione
		forwardMsg := tgbotapi.NewForward(config.DestChannelID, config.SourceChannelID, message.MessageID)
		sentMsg, err := bot.Send(forwardMsg)
		if err != nil {
			log.Printf("Error forwarding message: %v", err)
			continue
		}

		log.Printf("Message forwarded to channel %d (MessageID: %d)", config.DestChannelID, sentMsg.MessageID)

		if config.CommentTemplate == "" {
			continue
		}

		// Invia un commento nel gruppo di discussione
		commentMsg := tgbotapi.NewMessage(config.DiscussionGroupID, config.CommentTemplate)
		commentMsg.ParseMode = "Markdown"
		commentMsg.ReplyToMessageID = sentMsg.MessageID
		commentMsg.DisableNotification = true

		_, err = bot.Send(commentMsg)
		if err != nil {
			log.Printf("Error sending comment to discussion group: %v", err)
			continue
		}

		log.Printf("Comment sent to discussion group %d", config.DiscussionGroupID)
	}
}
