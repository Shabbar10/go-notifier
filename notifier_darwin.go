package notifier

/*
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdlib.h>

void requestAuthorization(const char *bundleID);
void deliverNotification(const char *notifID, const char *title, const char *message, const char *imagePath,
    const char **actionKeys, const char **actionLabels, int actionCount);
*/
import "C"
import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	callbacks    = make(map[string]func(string))
	callbacksMu sync.Mutex
	notifCounter atomic.Uint64
)

type darwinNotifier struct {
	authorized bool
}

// NewNotifier constructs notifier for macOS
func NewNotifier() (Notifier, error) {
	return &darwinNotifier{}, nil
}

//export onNotificationAction
func onNotificationAction(cNotifID *C.char, cActionID *C.char) {
	notifID := C.GoString(cNotifID)
	actionID := C.GoString(cActionID)

	// Ignore default tap and dismiss — only fire for custom action buttons
	if strings.HasPrefix(actionID, "com.apple.") {
		return
	}

	callbacksMu.Lock()
	cb, ok := callbacks[notifID]
	if ok {
		delete(callbacks, notifID)
	}
	callbacksMu.Unlock()

	if ok && cb != nil {
		cb(actionID)
	}
}

// DeliverNotification sends a notification via UNUserNotificationCenter
func (n *darwinNotifier) DeliverNotification(notification Notification) error {
	if !n.authorized {
		cBundleID := C.CString(notification.BundleID)
		C.requestAuthorization(cBundleID)
		C.free(unsafe.Pointer(cBundleID))
		n.authorized = true
	}

	id := notifCounter.Add(1)
	notifID := fmt.Sprintf("notif_%d", id)

	if len(notification.Actions) > 0 && notification.OnAction != nil {
		callbacksMu.Lock()
		callbacks[notifID] = notification.OnAction
		callbacksMu.Unlock()
	}

	cNotifID := C.CString(notifID)
	defer C.free(unsafe.Pointer(cNotifID))
	cTitle := C.CString(notification.Title)
	defer C.free(unsafe.Pointer(cTitle))
	cMessage := C.CString(notification.Message)
	defer C.free(unsafe.Pointer(cMessage))
	cImagePath := C.CString(notification.ImagePath)
	defer C.free(unsafe.Pointer(cImagePath))

	actionCount := len(notification.Actions)
	var cKeys **C.char
	var cLabels **C.char

	if actionCount > 0 {
		keys := make([]*C.char, actionCount)
		labels := make([]*C.char, actionCount)
		for i, a := range notification.Actions {
			keys[i] = C.CString(a.Key)
			labels[i] = C.CString(a.Label)
		}
		cKeys = &keys[0]
		cLabels = &labels[0]

		defer func() {
			for i := 0; i < actionCount; i++ {
				C.free(unsafe.Pointer(keys[i]))
				C.free(unsafe.Pointer(labels[i]))
			}
		}()
	}

	C.deliverNotification(cNotifID, cTitle, cMessage, cImagePath, cKeys, cLabels, C.int(actionCount))
	return nil
}
