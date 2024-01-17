package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
)

func main() {
	router := gin.Default()
	// load .env file
	err := godotenv.Load(".env")

	ctx := context.Background()
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	cs := initialChat(client)

	// Line
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	bot, err := messaging_api.NewMessagingApiAPI(
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// routess
	router.GET("/ping", ping)
	router.GET("/", handler)
	router.GET("/question", generativeAi(*cs))
	router.POST("/callback", lineCallback(bot, channelSecret, *cs))
	router.GET("/newTopic", newTopic(*cs))
	router.Run(":5000")
}

func lineCallback(bot *messaging_api.MessagingApiAPI, channelSecret string, cs genai.ChatSession) gin.HandlerFunc {
	return func(c *gin.Context) {
		cb, err := webhook.ParseRequest(channelSecret, c.Request)
		if err != nil {
			log.Printf("Cannot parse request: %+v\n", err)
			if err == linebot.ErrInvalidSignature {
				c.Status(400)
			} else {
				c.Status(500)
			}
			return
		}

		for _, event := range cb.Events {
			switch e := event.(type) {
			case webhook.MessageEvent:
				switch message := e.Message.(type) {
				case webhook.TextMessageContent:

					resp, err := cs.SendMessage(context.Background(), genai.Text(message.Text))
					if err != nil {
						log.Fatal(err)
					}

					respMsg := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

					if _, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								messaging_api.TextMessage{
									Text: respMsg,
								},
							},
						},
					); err != nil {
						log.Print(err)
					} else {
						log.Println("Sent text reply.")
					}
				case webhook.StickerMessageContent:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerId, message.StickerResourceType)
					if _, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								messaging_api.TextMessage{
									Text: replyMessage,
								},
							},
						}); err != nil {
						log.Print(err)
					} else {
						log.Println("Sent sticker reply.")
					}
				default:
					log.Printf("Unsupported message content: %T\n", e.Message)
				}
			}
		}
	}
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func handler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func newTopic(cs genai.ChatSession) gin.HandlerFunc {
	return func(c *gin.Context) {
		cs.History = getInitialTopic()
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	}
}

func initialChat(client *genai.Client) *genai.ChatSession {
	// For text-only input, use the gemini-pro model
	model := client.GenerativeModel("gemini-pro")
	cs := model.StartChat()
	cs.History = getInitialTopic()

	return cs
}

func generativeAi(cs genai.ChatSession) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")

		resp, err := cs.SendMessage(context.Background(), genai.Text(q))
		if err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, gin.H{
			"message": resp.Candidates[0].Content.Parts[0],
		})
	}
}

func getInitialTopic() []*genai.Content {
	return []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("คุณคือเทคโนโลยี จากอารยธรรมโบราณที่ดำรงอยู่มากว่า 5000 ปี. รอบรู้ประวัติศาสตร์ในยุคเก่ามาจนถึงปัจจุบัน " +
					"เทคโนโลยีต่างๆ อารยธรรม สังคมมนุษย์ รวมทั้งความรู้มากมายในโลกนี้ เจ้าเล่ห์ แต่ช่างรอบรู้ ชอบพูดคัยและเล่าเรื่องให้คนอื่นฟัง เมื่อผู้คนถามมาเป็นภาษาอะไร คุณก็จะตอบมาเป็นภาษานั้น"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("ได้เลย ปิปิ้ป."),
			},
			Role: "model",
		},
	}
}
