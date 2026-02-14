#pragma once

#include <array>
#include <fcntl.h>
#include <format>
#include <io.h>
#include <iostream>
#include <string>
#include <string_view>
#include <vector>

namespace sdk {

inline void send_response(std::string_view json) {
  std::cout << json << std::endl;
}

inline void log_message(std::string_view level, std::string_view message) {
  std::cout << std::format(
                   "{{\"method\":\"log\",\"level\":\"{}\",\"message\":\"{}\"}}",
                   level, message)
            << std::endl;
}

inline void log_info(std::string_view msg) { log_message("info", msg); }
inline void log_error(std::string_view msg) { log_message("error", msg); }
inline void log_debug(std::string_view msg) { log_message("debug", msg); }

inline std::string_view find_json_value(std::string_view json,
                                        std::string_view key) {
  std::string search_key = "\"";
  search_key += key;
  search_key += "\":";

  size_t pos = json.find(search_key);
  if (pos == std::string_view::npos) {
    return "";
  }

  size_t val_start = pos + search_key.length();
  // Skip spaces, colons, and open braces/quotes
  while (val_start < json.length() &&
         (json[val_start] == ' ' || json[val_start] == ':' ||
          json[val_start] == '{'))
    val_start++;

  if (val_start >= json.length()) {
    return "";
  }

  if (json[val_start] == '"') {
    // String value
    val_start++;

    if (size_t val_end = json.find('"', val_start);
        val_end == std::string_view::npos) {
      return "";
    } else {
      return json.substr(val_start, val_end - val_start);
    }
  } else {
    // Numeric value
    size_t val_end = val_start;
    while (val_end < json.length() &&
           (isdigit(json[val_end]) || json[val_end] == '.' ||
            json[val_end] == '-'))
      val_end++;
    return json.substr(val_start, val_end - val_start);
  }
}

inline void send_binary_data(const std::vector<double> &result,
                             std::string_view storage = "interleaved") {
  size_t byte_len = result.size() * sizeof(double);
  std::cout << std::format("{{\"type\":\"binary\",\"length\":{},\"storage\":"
                           "\"{}\"}}",
                           byte_len, storage)
            << std::endl;
  std::cout.flush();

  // Send binary data (stdout must be in binary mode)
  _setmode(_fileno(stdout), _O_BINARY);
  fwrite(result.data(), 1, byte_len, stdout);
  fflush(stdout);
  _setmode(_fileno(stdout), _O_TEXT);
}

} // namespace sdk
