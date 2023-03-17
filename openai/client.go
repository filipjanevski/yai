package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ekkinox/hey/detect"
	"io"
	"net/http"
	"strings"

	"github.com/ekkinox/hey/config"
)

const error_message = "An error occurred."
const error_flag = "GENERR"

type Response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	Config config.Config
}

func InitClient(cfg config.Config) (*Client, error) {
	if cfg.OpenAI.Key == "" || cfg.OpenAI.Key == config.Openai_Key_Placeholder {
		return nil, fmt.Errorf("openai api key is not defined")
	}

	return &Client{cfg}, nil
}

func (c *Client) Send(input string) (bool, string, error) {

	payload := fmt.Sprintf(
		`{"model":"%s","messages":[{"role":"system","content":"%s"},{"role":"user","content":"%s"}]}`,
		c.Config.OpenAI.Model,
		c.buildSystemPrompt(),
		input,
	)

	req, err := http.NewRequest(http.MethodPost, c.Config.OpenAI.Url, bytes.NewReader([]byte(payload)))
	if err != nil {
		return false, error_message, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Config.OpenAI.Key))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("openai: could not make request")
		return false, error_message, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("openai: could not read response body")
		return false, error_message, err
	}

	var data Response
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("openai: could not unmarshal JSON")
		return false, error_message, err
	}

	output := strings.Trim(data.Choices[0].Message.Content, "\n")

	if strings.Contains(output, error_flag) {
		output = strings.ReplaceAll(output, error_flag, "")
		return false, output, nil
	}

	return true, output, nil
}

func (c *Client) buildSystemPrompt() string {
	prompt := "You are a helpful command line assistant. "
	prompt += "You will ONLY generate commands based on user input. "

	if c.Config.System.OperatingSystem != detect.OS_other {
		prompt += fmt.Sprintf("The operating system is %s. ", c.Config.System.OperatingSystem)
	}

	if c.Config.System.Distribution != "" {
		prompt += fmt.Sprintf("The distribution is %s. ", c.Config.System.Distribution)
	}

	if c.Config.System.Shell != "" {
		prompt += fmt.Sprintf("The shell is %s. ", c.Config.System.Shell)
	}

	if c.Config.System.OperatingSystem != "" {
		prompt += fmt.Sprintf("The home directory is %s. ", c.Config.System.HomeDir)
	}

	prompt += "Your response should contain ONLY the command and NO explanation. "
	prompt += "Do NOT ever use newlines to separate commands, instead use ; or &&. "
	prompt += fmt.Sprintf("If your response does not return a command I can execute, ALWAYS add %s at the beginning of you response.", error_flag)

	return prompt
}