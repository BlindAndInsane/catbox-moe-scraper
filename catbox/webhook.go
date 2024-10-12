package catbox

import (
	"log"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/webhook"
)

var (
	webhook_client webhook.Client
)

func init() {
	client, err := webhook.NewWithURL(G_config.WebhookURL)
	if err != nil {
		G_logger.Errorln(err)
	}
	webhook_client = client
}

func SendMessageToWebhook(message string) error {
	msg, err := webhook_client.CreateMessage(discord.NewWebhookMessageCreateBuilder().
		SetContent(message).
		Build(),
	)
	if err != nil {
		log.Printf("Failed to send webhook message: %v", err)
		return err
	}

	log.Printf("Webhook message sent: %s", msg.Content)
	return nil
}
