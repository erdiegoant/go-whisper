package notify

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework UserNotifications -framework Foundation
// #import <UserNotifications/UserNotifications.h>
// #import <Foundation/Foundation.h>
// #include <stdlib.h>
//
// void show_notification(const char *title, const char *body) {
//     NSString *nsTitle = [NSString stringWithUTF8String:title];
//     NSString *nsBody  = [NSString stringWithUTF8String:body];
//
//     dispatch_async(dispatch_get_main_queue(), ^{
//         UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
//
//         [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
//                               completionHandler:^(BOOL granted, NSError *error) {
//             if (!granted) return;
//
//             UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
//             content.title = nsTitle;
//             content.body  = nsBody;
//
//             NSString *identifier = [[NSUUID UUID] UUIDString];
//             UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:identifier
//                                                                                   content:content
//                                                                                   trigger:nil];
//             [center addNotificationRequest:request withCompletionHandler:nil];
//         }];
//     });
// }
import "C"

import "unsafe"

// Show displays a macOS notification attributed to GoWhisper.app.
// body is truncated to 100 characters. Non-blocking — errors are silently ignored.
func Show(title, body string) {
	if len(body) > 100 {
		body = body[:100]
	}
	cTitle := C.CString(title)
	cBody := C.CString(body)
	go func() {
		C.show_notification(cTitle, cBody)
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cBody))
	}()
}
