package transcribe

// #include <whisper.h>
// #include <stdlib.h>
//
// // No-op log callback — silences all whisper.cpp stderr output.
// static void noop_log_callback(enum ggml_log_level level, const char* text, void* user_data) {}
//
// void suppress_whisper_logs(void) {
//     whisper_log_set(noop_log_callback, NULL);
// }
import "C"

// SuppressLogs silences all whisper.cpp diagnostic output.
// Call once before loading any model.
func SuppressLogs() {
	C.suppress_whisper_logs()
}
