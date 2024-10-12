package catbox

import (
	"log"

	"github.com/disgoorg/disgo/discord"
)

func SendMessageToWebhook(message string) error {
	msg, err := G_webhook_client.CreateMessage(discord.NewWebhookMessageCreateBuilder().
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
