package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// TODO: move to a diffrent file
type Assistant struct {
	ID              string         `json:"id"`
	Object          string         `json:"object"`
	CreatedAt       int64          `json:"created_at"`
	Name            string         `json:"name"`
	Description     *string        `json:"description"` // nullable
	Model           string         `json:"model"`
	Instructions    string         `json:"instructions"`
	Tools           []Tool         `json:"tools"`
	TopP            float64        `json:"top_p"`
	Temperature     float64        `json:"temperature"`
	ReasoningEffort *string        `json:"reasoning_effort"` // nullable
	ToolResources   ToolResources  `json:"tool_resources"`
	Metadata        map[string]any `json:"metadata"`
	ResponseFormat  string         `json:"response_format"`
	Error           OpenAIerr      `json:"error"`
}

type OpenAIerr struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"` // null value maps to a pointer
	Code    string  `json:"code"`
}

type Tool struct {
	Type string `json:"type"`
}

type ToolResources struct {
	CodeInterpreter CodeInterpreter `json:"code_interpreter"`
}

type CodeInterpreter struct {
	FileIDs []string `json:"file_ids"`
}

type Thread struct {
	ID            string         `json:"id"`
	Object        string         `json:"object"`
	CreatedAt     int64          `json:"created_at"`
	Metadata      map[string]any `json:"metadata"`
	ToolResources map[string]any `json:"tool_resources"`
}

type Run struct {
	ID string `json:"id"`
}

type OpenAiDataAccess struct {
	OpenAIKey string
	InfoLog   *log.Logger
	ErrorLog  *log.Logger
	ApiLog    *log.Logger
}

// {
//   "object": "list",
//   "data": [
//     {
//       "id": "msg_eeDrybmCXRJBaCMSaIeN9Rbx",
//       "object": "thread.message",
//       "created_at": 1746319599,
//       "assistant_id": "asst_vlQvXuyCw7gtXrMQHm8KiKSa",
//       "thread_id": "thread_fVyKWBpSJZISBNMl8MiDUtpm",
//       "run_id": "run_amOP2sNAPaExW2sSbSsRYEwh",
//       "role": "assistant",
//       "content": [
//         {
//           "type": "text",
//           "text": {
//             "value": "Certainly, Jane Doe! The solution to the equation \\(3x + 11 = 14\\) is \\(x = 1\\).",
//             "annotations": []
//           }
//         }
//       ],
//       "attachments": [],
//       "metadata": {}
//     },
//     {
//       "id": "msg_JFkarkJUWFJSYUOrWcSyxoHn",
//       "object": "thread.message",
//       "created_at": 1746318988,
//       "assistant_id": null,
//       "thread_id": "thread_fVyKWBpSJZISBNMl8MiDUtpm",
//       "run_id": null,
//       "role": "user",
//       "content": [
//         {
//           "type": "text",
//           "text": {
//             "value": "I need to solve the equation `3x + 11 = 14`. Can you help me?",
//             "annotations": []
//           }
//         }
//       ],
//       "attachments": [],
//       "metadata": {}
//     }
//   ],
//   "first_id": "msg_eeDrybmCXRJBaCMSaIeN9Rbx",
//   "last_id": "msg_JFkarkJUWFJSYUOrWcSyxoHn",
//   "has_more": false
// }

type MessageList struct {
	Object  string    `json:"object"`
	Data    []Message `json:"data"`
	FirstID string    `json:"first_id"`
	LastID  string    `json:"last_id"`
	HasMore bool      `json:"has_more"`
	Error   OpenAIerr `json:"error"`
}

type Message struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	CreatedAt   int64          `json:"created_at"`
	AssistantID *string        `json:"assistant_id"` // nullable
	ThreadID    string         `json:"thread_id"`
	RunID       *string        `json:"run_id"` // nullable
	Role        string         `json:"role"`
	Content     []ContentBlock `json:"content"`
	Attachments []any          `json:"attachments"` // assuming attachments can vary
	Metadata    map[string]any `json:"metadata"`
}

type ContentBlock struct {
	Type string    `json:"type"`
	Text TextBlock `json:"text"`
}

type TextBlock struct {
	Value       string `json:"value"`
	Annotations []any  `json:"annotations"` // adjust type if known
}

var OpenAiError = fmt.Errorf("OPEN API Error")

func (da OpenAiDataAccess) createAssistant(name string, instructions string) (Assistant, error) {
	reqBody := bytes.NewBufferString(fmt.Sprintf(`{
		    "instructions": "%s",
		    "name": "%s",
		    "tools": [{"type": "code_interpreter"}],
		    "model": "gpt-4o"
		  }`, instructions, name))

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/assistants", reqBody)
	if err != nil {
		// fmt.Println("Error creating request:", err)
		return Assistant{}, err
	}

	da.setDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// fmt.Println("Error:", err)
		return Assistant{}, err
	}
	//defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("Error reading body:", err)
		return Assistant{}, err
	}
	da.ApiLog.Println(string(respBody))
	_ = resp.Body.Close()

	var assistantData Assistant
	err = json.Unmarshal(respBody, &assistantData)
	if err != nil {
		// fmt.Println("Error unmarshaling:", err)
		return Assistant{}, err
	}

	if assistantData.Error.Message != "" {
		//fmt.Println("API Error:", msg)
		return assistantData, fmt.Errorf("error creating assistant with error \"%s\": %w", assistantData.Error.Message, OpenAiError)
	}

	return assistantData, nil
}

func (da OpenAiDataAccess) createThread() (Thread, error) {
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/threads", nil)
	if err != nil {
		// fmt.Println("Error creating request:", err)
		return Thread{}, err
	}

	da.setDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// fmt.Println("Error:", err)
		return Thread{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("Error reading body:", err)
		return Thread{}, err
	}
	da.ApiLog.Println(string(respBody))
	_ = resp.Body.Close()

	var thread Thread
	err = json.Unmarshal(respBody, &thread)
	if err != nil {
		// fmt.Println("Error unmarshaling:", err)
		return Thread{}, err
	}
	return thread, nil
}

func (da OpenAiDataAccess) RefreshThread(thread Thread) (MessageList, error) {
	reqUrl := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", thread.ID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		// fmt.Println("Error creating request:", err)
		return MessageList{}, err
	}

	da.setDefaultHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// fmt.Println("Error:", err)
		return MessageList{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("Error reading body:", err)
		return MessageList{}, err
	}
	da.ApiLog.Println(string(respBody))
	_ = resp.Body.Close()

	var newThread MessageList
	err = json.Unmarshal(respBody, &newThread)
	if err != nil {
		// fmt.Println("Error unmarshaling:", err)
		return MessageList{}, err
	}
	return newThread, nil
}

func (da OpenAiDataAccess) AddThreadMessage(thread Thread, role string, content string) (Message, error) {
	reqUrl := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", thread.ID)
	reqBody := bytes.NewBufferString(fmt.Sprintf(`{
			"role": "%s",
			"content": "%s"
		  }`, role, content))

	req, err := http.NewRequest("POST", reqUrl, reqBody)
	if err != nil {
		// fmt.Println("Error creating request:", err)
		return Message{}, err
	}

	da.setDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// fmt.Println("Error:", err)
		return Message{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	da.ApiLog.Println(string(respBody))

	if err != nil {
		// fmt.Println("Error reading body:", err)
		return Message{}, err
	}
	_ = resp.Body.Close()

	var newThread Message
	err = json.Unmarshal(respBody, &newThread)
	if err != nil {
		fmt.Printf("Error unmarshaling response: %s\n", string(respBody))
		return Message{}, err
	}
	return newThread, nil
}

func (da OpenAiDataAccess) RunAssistant(thread Thread, assistant Assistant, instructions string) (Run, error) {
	reqUrl := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", thread.ID)
	reqBody := bytes.NewBufferString(fmt.Sprintf(`{
			"assistant_id": "%s",
			"instructions": "%s"
		  }`, assistant.ID, instructions))

	req, err := http.NewRequest("POST", reqUrl, reqBody)
	if err != nil {
		// fmt.Println("Error creating request:", err)
		return Run{}, err
	}

	da.setDefaultHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// fmt.Println("Error:", err)
		return Run{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("Error reading body:", err)
		return Run{}, err
	}
	da.ApiLog.Println(string(respBody))
	_ = resp.Body.Close()

	var newRun Run
	err = json.Unmarshal(respBody, &newRun)
	if err != nil {
		// fmt.Println("Error unmarshaling:", err)
		return Run{}, err
	}
	return newRun, nil
}

func (da OpenAiDataAccess) setDefaultHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", da.OpenAIKey))
	req.Header.Set("OpenAI-Beta", "assistants=v2")
}

// //create thread
// curl https://api.openai.com/v1/threads \
//   -H "Content-Type: application/json" \
//   -H "Authorization: Bearer $OPENAI_API_KEY" \
//   -H "OpenAI-Beta: assistants=v2" \
//   -d ''
//

// curl https://api.openai.com/v1/threads/thread_uf0eskE80SvqNHcTmho6UHAV/messages \
//   -H "Content-Type: application/json" \
//   -H "Authorization: Bearer $OPENAI_API_KEY" \
//   -H "OpenAI-Beta: assistants=v2" \
//   -d '{
//       "role": "user",
//       "content": "create a orc"
//     }'
//

// curl https://api.openai.com/v1/threads/thread_uf0eskE80SvqNHcTmho6UHAV/runs
//   -H "Authorization: Bearer $OPENAI_API_KEY"
//   -H "Content-Type: application/json"
//   -H "OpenAI-Beta: assistants=v2"
//   -d '{
//     "assistant_id": "asst_SjuZ4ZcIHfbovT7yHV7zAkwL",
//     "instructions": "Please address the user as Jane Doe. The user has a premium account."
