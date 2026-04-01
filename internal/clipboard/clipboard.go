package clipboard

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreGraphics
// #import <AppKit/AppKit.h>
// #include <ApplicationServices/ApplicationServices.h>
// #include <stdlib.h>
//
// // Synchronously write text to the general pasteboard.
// void set_clipboard(const char *text) {
//     NSPasteboard *pb = [NSPasteboard generalPasteboard];
//     [pb clearContents];
//     [pb setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString];
// }
//
// // Synchronously read text from the general pasteboard.
// // Returns a malloc'd C string the caller must free, or NULL if empty.
// char* get_clipboard(void) {
//     NSPasteboard *pb = [NSPasteboard generalPasteboard];
//     NSString *str = [pb stringForType:NSPasteboardTypeString];
//     if (str == nil) return NULL;
//     const char *utf8 = [str UTF8String];
//     return utf8 ? strdup(utf8) : NULL;
// }
//
// void simulate_paste(void) {
//     CGEventRef down = CGEventCreateKeyboardEvent(NULL, 9, true);
//     CGEventSetFlags(down, kCGEventFlagMaskCommand);
//     CGEventPost(kCGHIDEventTap, down);
//     CFRelease(down);
//
//     CGEventRef up = CGEventCreateKeyboardEvent(NULL, 9, false);
//     CGEventSetFlags(up, kCGEventFlagMaskCommand);
//     CGEventPost(kCGHIDEventTap, up);
//     CFRelease(up);
// }
import "C"

import (
	"time"
	"unsafe"
)

// Write puts text on the clipboard without simulating a paste or disturbing
// whatever was there before. Use this when the caller just wants text available
// to paste manually (e.g. copying from history).
func Write(text string) {
	cText := C.CString(text)
	C.set_clipboard(cText)
	C.free(unsafe.Pointer(cText))
}

// Paste writes text to the clipboard, simulates Cmd+V into the active window,
// then restores the previous clipboard contents. Safe to call from any goroutine.
func Paste(text string) error {
	// Save current clipboard contents.
	prevPtr := C.get_clipboard()
	var prev string
	if prevPtr != nil {
		prev = C.GoString(prevPtr)
		C.free(unsafe.Pointer(prevPtr))
	}

	// Write transcribed text synchronously.
	cText := C.CString(text)
	C.set_clipboard(cText)
	C.free(unsafe.Pointer(cText))

	// Brief pause so the target app sees the updated clipboard before the paste event.
	time.Sleep(50 * time.Millisecond)

	// Simulate Cmd+V.
	C.simulate_paste()

	// Give the paste event time to be processed before restoring the clipboard.
	time.Sleep(150 * time.Millisecond)

	// Restore previous contents.
	if prevPtr != nil {
		cPrev := C.CString(prev)
		C.set_clipboard(cPrev)
		C.free(unsafe.Pointer(cPrev))
	}

	return nil
}
