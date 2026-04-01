package notify

// #cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
// #cgo LDFLAGS: -framework Foundation
// #import <Foundation/Foundation.h>
// #include <stdlib.h>
//
// void show_notification(const char *title, const char *body) {
//     NSUserNotification *notification = [[NSUserNotification alloc] init];
//     notification.title           = [NSString stringWithUTF8String:title];
//     notification.informativeText = [NSString stringWithUTF8String:body];
//
//     dispatch_async(dispatch_get_main_queue(), ^{
//         [[NSUserNotificationCenter defaultUserNotificationCenter] deliverNotification:notification];
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
