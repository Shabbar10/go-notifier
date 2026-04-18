package notifier

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsNotifier struct{}

// NewNotifier constructs notifier for Windows
func NewNotifier() (Notifier, error) {
	return &windowsNotifier{}, nil
}

// DeliverNotification sends a toast notification via PowerShell
func (n *windowsNotifier) DeliverNotification(notification Notification) error {
	// Build the toast XML
	var actionsXML string
	if len(notification.Actions) > 0 {
		var buttons []string
		for _, a := range notification.Actions {
			buttons = append(buttons, fmt.Sprintf(
				`<action content="%s" arguments="%s" activationType="foreground"/>`,
				escapeXML(a.Label), escapeXML(a.Key),
			))
		}
		actionsXML = "<actions>" + strings.Join(buttons, "") + "</actions>"
	}

	var imageXML string
	if notification.ImagePath != "" {
		imageXML = fmt.Sprintf(` <image placement="appLogoOverride" src="%s"/>`, escapeXML(notification.ImagePath))
	}

	toastXML := fmt.Sprintf(`<toast>
  <visual>
    <binding template="ToastGeneric">
      <text>%s</text>
      <text>%s</text>%s
    </binding>
  </visual>
  %s
</toast>`, escapeXML(notification.Title), escapeXML(notification.Message), imageXML, actionsXML)

	// PowerShell script to show the toast and optionally capture the action
	var ps string
	if len(notification.Actions) > 0 && notification.OnAction != nil {
		// With actions: register for activation and wait
		ps = fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml('%s')
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$event = Register-ObjectEvent -InputObject $toast -EventName Activated -Action {
    $args = $Event.SourceEventArgs
    Write-Host $args.Arguments
}
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe")
$notifier.Show($toast)
Wait-Event -Timeout 30 | Out-Null
`, strings.ReplaceAll(toastXML, "'", "''"))
	} else {
		// Simple notification without waiting
		ps = fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml('%s')
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe")
$notifier.Show($toast)
`, strings.ReplaceAll(toastXML, "'", "''"))
	}

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)

	if len(notification.Actions) > 0 && notification.OnAction != nil {
		go func() {
			out, err := cmd.Output()
			if err != nil {
				return
			}
			actionKey := strings.TrimSpace(string(out))
			if actionKey != "" {
				notification.OnAction(actionKey)
			}
		}()
		return nil
	}

	return cmd.Run()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
