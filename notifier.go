// Copyright 2016 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package notifier

// Action defines a notification button
type Action struct {
	Key   string // Identifier returned in the callback
	Label string // Text displayed on the button
}

// Notification defines a notification
type Notification struct {
	Title     string
	Message   string
	ImagePath string
	Actions   []Action
	OnAction  func(actionKey string) // Called when the user clicks an action button

	// For darwin
	Timeout  float64
	BundleID string

	// For windows
	ToastPath string // Path to toast.exe
}

// Notifier knows how to deliver a notification
type Notifier interface {
	DeliverNotification(notification Notification) error
}
