package data

import (
	"errors"
	"flag"
	"testing"
)

var openAiKey string

func init() {
	flag.StringVar(&openAiKey, "open-ai-key", "", "OpenAI API key")
}

// dependednt on openai api
func TestCreateAssistant(t *testing.T) {
	assist, err := openAiDa.createAssistant("test", "Please respond with the numbers 1234 and nothing else.")
	if err != nil {
		t.Fatal(err)
	}
	if assist.Instructions != "Please respond with the numbers 1234 and nothing else." {
		t.Fatal(err)
	}
}

func TestCreateAssistantNoKey(t *testing.T) {
	da := openAiDa
	da.OpenAIKey = "1234"
	assist, err := da.createAssistant("test", "Please respond with the numbers 1234 and nothing else.")
	if !errors.Is(err, OpenAiError) {
		t.Fatal(err)
	}
	if assist.Error.Code != "invalid_api_key" {
		t.Fatal(err)
	}
}

// func TestCreateMessage(t *testing.T) {
// 	t.Parallel()
//
// 	da := OpenAiDataAccess{OpenAIKey: openAiKey}
// 	assist, err := da.createAssistant("test", "Please respond with the numbers 1234 and nothing else.")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	userMessage := "hello how are you?"
//
// 	thread, err := da.createThread()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	mesg, err := da.AddThreadMessage(thread, "user", userMessage)
// 	t.Log(mesg)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if mesg.Content[0].Text.Value != userMessage {
// 		t.Fatal(err)
// 	}
// 	if thread.ID == "" {
// 		t.Fatal(err)
// 	}
//
// 	_, err = da.RunAssistant(thread, assist, "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	time.Sleep(10 * time.Second) //TODO: substitute with poll or something
// 	mesgList, err := da.RefreshThread(thread)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(mesgList.Data) != 2 {
// 		t.Fatal(err)
// 	}
// 	if mesgList.Data[0].Content[0].Text.Value != "1234" {
// 		t.Fatal(err)
// 	}
// 	// t.Log(fmt.Sprintf("%+v", mesgList))
// }
