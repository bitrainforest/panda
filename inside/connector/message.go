// Note: this package is not used now, just for websocket in future!
package connector

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bitrainforest/PandaAgent/pkg/util"
)

type MessageType string

const (
	FileType    MessageType = "file"
	CommandType MessageType = "command"
	JsonType    MessageType = "json"
)

type Module string

const (
	PythonModule Module = "python"
	ShellModule  Module = "shell"
	FetchModule  Module = "fetch"
	CopyModule   Module = "copy"
)

type Message struct {
	Type      MessageType
	Timestamp string
	Module    Module
	Session   string
	Payload   json.RawMessage
}

func NewMessage(typ MessageType, mod Module, data []byte) *Message {
	m := &Message{
		Type:      typ,
		Timestamp: time.Now().Format(util.TimeFormat),
		Module:    mod,
		Payload:   make([]byte, len(data)),
	}
	copy(m.Payload, data)
	return m
}

type MessageBytes []byte

func (mb MessageBytes) Deserialize() (Message, error) {
	var m Message
	err := json.Unmarshal([]byte(mb), &m)
	return m, err
}

func (m *Message) Serialize() ([]byte, error) {
	return json.Marshal(*m)
}

type MessageError struct {
	Err error
	Msg *Message
}

type JsonResponse struct {
	Out string `json:"output"`
	Err string `json:"output_error"`
}

func NewJsonMessage(data, err string) *Message {
	ret := &JsonResponse{
		Out: data,
		Err: fmt.Sprintf("%s", err),
	}
	retBytes, _ := json.Marshal(ret)
	return NewMessage(JsonType, "", retBytes)
}
