//go:build windows

package main

import (
	"log"

	"git.sr.ht/~jackmordaunt/go-toast/v2"
)

// toastNotifier 用 Windows Toast 发送通知。
type toastNotifier struct{}

func (toastNotifier) Notify(title, body string) {
	n := toast.Notification{
		AppID: "workbuddy-checkin",
		Title: title,
		Body:  body,
	}
	if err := n.Push(); err != nil {
		log.Printf("[notify] push failed: %v", err)
	}
}

func newNotifier() Notifier { return toastNotifier{} }
