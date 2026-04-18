package notifier

import (
	"os/exec"
	"strings"
)

type windowsNotifier struct{}

// NewNotifier constructs notifier for Windows
func NewNotifier() (Notifier, error) {
	return &windowsNotifier{}, nil
}

// DeliverNotification sends a toast notification via the toaster helper
func (n *windowsNotifier) DeliverNotification(notification Notification) error {
	args := []string{
		"--title", notification.Title,
		"--message", notification.Message,
	}

	if len(notification.Actions) > 0 {
		var keys, labels []string
		for _, a := range notification.Actions {
			keys = append(keys, a.Key)
			labels = append(labels, a.Label)
		}
		args = append(args, "--action-keys", strings.Join(keys, ","))
		args = append(args, "--action-labels", strings.Join(labels, ","))
	}

	if notification.ImagePath != "" {
		args = append(args, "--image", notification.ImagePath)
	}

	toasterPath := notification.ToastPath
	if toasterPath == "" {
		toasterPath = "toaster.exe"
	}

	cmd := exec.Command(toasterPath, args...)

	if len(notification.Actions) > 0 && notification.OnAction != nil {
		go func() {
			out, err := cmd.Output()
			if err != nil {
				return
			}
			actionKey := strings.TrimSpace(string(out))
			if actionKey != "" && !strings.HasPrefix(actionKey, "__") {
				notification.OnAction(actionKey)
			}
		}()
		return nil
	}

	return cmd.Run()
}
