package notifier

import (
	"fmt"

	dbus "github.com/godbus/dbus/v5"
)

const (
	dbusNotifDest  = "org.freedesktop.Notifications"
	dbusNotifPath  = "/org/freedesktop/Notifications"
	dbusNotifIface = "org.freedesktop.Notifications"
)

type linuxNotifier struct {
	conn *dbus.Conn
}

// NewNotifier constructs notifier for Linux
func NewNotifier() (Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("could not connect to session bus: %w", err)
	}
	return &linuxNotifier{conn: conn}, nil
}

// DeliverNotification sends a notification via D-Bus
func (n *linuxNotifier) DeliverNotification(notification Notification) error {
	// Build the actions slice: alternating (key, label) pairs
	var actions []string
	for _, a := range notification.Actions {
		actions = append(actions, a.Key, a.Label)
	}

	obj := n.conn.Object(dbusNotifDest, dbusNotifPath)
	call := obj.Call(dbusNotifIface+".Notify", 0,
		"Synchronum",                      // app_name
		uint32(0),                         // replaces_id
		notification.ImagePath,            // app_icon
		notification.Title,                // summary
		notification.Message,              // body
		actions,                           // actions
		map[string]dbus.Variant{},         // hints
		int32(-1),                         // expire_timeout (-1 = server default)
	)
	if call.Err != nil {
		return fmt.Errorf("could not send notification: %w", call.Err)
	}

	var notifID uint32
	if err := call.Store(&notifID); err != nil {
		return fmt.Errorf("could not read notification ID: %w", err)
	}

	// If there are actions and a callback, listen for the user's response
	if len(notification.Actions) > 0 && notification.OnAction != nil {
		go n.listenForAction(notifID, notification.OnAction)
	}

	return nil
}

func (n *linuxNotifier) listenForAction(notifID uint32, onAction func(string)) {
	if err := n.conn.AddMatchSignal(
		dbus.WithMatchInterface(dbusNotifIface),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		return
	}

	signals := make(chan *dbus.Signal, 1)
	n.conn.Signal(signals)
	defer n.conn.RemoveSignal(signals)

	for sig := range signals {
		if sig.Name != dbusNotifIface+".ActionInvoked" {
			continue
		}
		if len(sig.Body) < 2 {
			continue
		}
		id, ok := sig.Body[0].(uint32)
		if !ok || id != notifID {
			continue
		}
		actionKey, ok := sig.Body[1].(string)
		if !ok {
			continue
		}
		onAction(actionKey)
		return
	}
}
