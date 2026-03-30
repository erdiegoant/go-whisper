package hotkey

// #cgo LDFLAGS: -framework ApplicationServices
// #include <ApplicationServices/ApplicationServices.h>
//
// int is_accessibility_trusted() {
//     CFStringRef keys[]   = { kAXTrustedCheckOptionPrompt };
//     CFBooleanRef values[] = { kCFBooleanTrue };
//     CFDictionaryRef options = CFDictionaryCreate(
//         kCFAllocatorDefault,
//         (const void **)keys,
//         (const void **)values,
//         1,
//         &kCFTypeDictionaryKeyCallBacks,
//         &kCFTypeDictionaryValueCallBacks
//     );
//     int trusted = (int)AXIsProcessTrustedWithOptions(options);
//     CFRelease(options);
//     return trusted;
// }
import "C"

// CheckAccessibility returns true if the app has Accessibility permission.
// If not, it triggers the macOS prompt to open System Settings > Accessibility.
func CheckAccessibility() bool {
	return C.is_accessibility_trusted() != 0
}
