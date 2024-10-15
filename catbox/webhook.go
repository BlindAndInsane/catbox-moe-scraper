package catbox

import (
	"github.com/disgoorg/disgo/discord"
)

func SendMessageToWebhook(message string) error {
	msg, err := G_webhook_client.CreateMessage(discord.NewWebhookMessageCreateBuilder().
		SetContent(message).
		Build(),
	)
	if err != nil {
		return err
	}

	G_logger.Debugf("Webhook message sent: %s", msg.Content)
	return nil
}
