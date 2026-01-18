package main

import (
	"encoding/json"
	"log"
	"time"
)

type auditEvent struct {
	Ts      string `json:"ts"`
	User    string `json:"user"`
	Remote  string `json:"remote"`
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	Target  string `json:"target,omitempty"`
	Bytes   int64  `json:"bytes,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func audit(user, remote, action, path, target string, bytes int64, err error) {
	ev := auditEvent{
		Ts:      time.Now().UTC().Format(time.RFC3339Nano),
		User:    user,
		Remote:  remote,
		Action:  action,
		Path:    path,
		Target:  target,
		Bytes:   bytes,
		Success: err == nil,
	}
	if err != nil {
		ev.Success = false
		ev.Error = err.Error()
	} else {
		ev.Success = true
	}
	b, _ := json.Marshal(ev)
	log.Println(string(b))
}

