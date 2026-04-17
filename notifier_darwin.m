#import <UserNotifications/UserNotifications.h>
#import <objc/runtime.h>

// Bundle identifier swizzling for non-bundled apps
static NSString *_fakeBundleIdentifier = nil;

@implementation NSBundle (FakeBundleIdentifier)
- (NSString *)__bundleIdentifier {
    if (self == [NSBundle mainBundle]) {
        return _fakeBundleIdentifier ? _fakeBundleIdentifier : @"com.synchronum.notifier";
    }
    return [self __bundleIdentifier];
}
@end

static void installFakeBundleIdentifierHook() {
    Class class = objc_getClass("NSBundle");
    if (class) {
        method_exchangeImplementations(
            class_getInstanceMethod(class, @selector(bundleIdentifier)),
            class_getInstanceMethod(class, @selector(__bundleIdentifier))
        );
    }
}

// Go callback declared in notifier_darwin.go
extern void onNotificationAction(const char *notifID, const char *actionID);

@interface NotifDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation NotifDelegate

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
    didReceiveNotificationResponse:(UNNotificationResponse *)response
    withCompletionHandler:(void (^)(void))completionHandler {

    const char *notifID = [response.notification.request.identifier UTF8String];
    const char *actionID = [response.actionIdentifier UTF8String];
    onNotificationAction(notifID, actionID);
    completionHandler();
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
    willPresentNotification:(UNNotification *)notification
    withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
    completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionSound);
}

@end

static NotifDelegate *sharedDelegate = nil;
static NSMutableSet<UNNotificationCategory *> *registeredCategories = nil;

void requestAuthorization(const char *bundleID) {
    // Only swizzle bundle ID if explicitly provided (for non-bundled apps)
    if (bundleID && strlen(bundleID) > 0) {
        _fakeBundleIdentifier = [NSString stringWithUTF8String:bundleID];
        installFakeBundleIdentifierHook();
    }

    UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];

    if (!sharedDelegate) {
        sharedDelegate = [[NotifDelegate alloc] init];
        center.delegate = sharedDelegate;
    }

    if (!registeredCategories) {
        registeredCategories = [NSMutableSet set];
    }

    [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
        completionHandler:^(BOOL granted, NSError *error) {
            if (error) {
                NSLog(@"Notification auth error: %@", error);
            }
            if (granted) {
                NSLog(@"Notification permission granted");
            } else {
                NSLog(@"Notification permission denied");
            }
        }];
}

void deliverNotification(const char *notifID, const char *title, const char *message, const char *imagePath,
    const char **actionKeys, const char **actionLabels, int actionCount) {

    UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
    content.title = [NSString stringWithUTF8String:title];
    content.body = [NSString stringWithUTF8String:message];
    content.sound = [UNNotificationSound defaultSound];

    if (imagePath && strlen(imagePath) > 0) {
        NSURL *imageURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:imagePath]];
        NSError *error = nil;
        UNNotificationAttachment *attachment = [UNNotificationAttachment
            attachmentWithIdentifier:@"image"
            URL:imageURL
            options:nil
            error:&error];
        if (attachment) {
            content.attachments = @[attachment];
        }
    }

    if (actionCount > 0) {
        NSMutableArray<UNNotificationAction *> *actions = [NSMutableArray array];
        for (int i = 0; i < actionCount; i++) {
            UNNotificationAction *action = [UNNotificationAction
                actionWithIdentifier:[NSString stringWithUTF8String:actionKeys[i]]
                title:[NSString stringWithUTF8String:actionLabels[i]]
                options:UNNotificationActionOptionNone];
            [actions addObject:action];
        }

        NSString *categoryID = [NSString stringWithFormat:@"cat_%s", notifID];
        UNNotificationCategory *category = [UNNotificationCategory
            categoryWithIdentifier:categoryID
            actions:actions
            intentIdentifiers:@[]
            options:0];

        [registeredCategories addObject:category];

        UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
        [center setNotificationCategories:registeredCategories];
        content.categoryIdentifier = categoryID;
    }

    NSString *identifier = [NSString stringWithUTF8String:notifID];
    UNNotificationRequest *request = [UNNotificationRequest
        requestWithIdentifier:identifier
        content:content
        trigger:nil];

    UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
    [center addNotificationRequest:request withCompletionHandler:^(NSError *error) {
        if (error) {
            NSLog(@"Notification delivery error: %@", error);
        }
    }];
}
