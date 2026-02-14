#ifndef UNICODE
#define UNICODE
#endif

#include <algorithm>
#include <cmath>
#include <ctime>
#include <fcntl.h>
#include <format>
#include <io.h>
#include <iomanip>
#include <iostream>
#include <random>
#include <string>
#include <string_view>
#include <vector>
#include <windows.h>

#pragma comment(linker, "/SUBSYSTEM:windows /ENTRY:mainCRTStartup")

#include "../../sdk/cpp/protocol.hpp"

// Global configuration
struct Config {
  int numPoints = 1000000;
  int numSeries = 3;
  int order = 6;
  double multiplier = 1.0;
  double noise = 1.0;
};

static Config g_config;

// Plugin metadata
constexpr std::string_view pluginName = "Random Walk Generator";
constexpr int pluginVersion = 1;

// Form schema for host-controlled UI
constexpr std::string_view formSchema = R"({
    "method": "show_form",
    "title": "Random Walk Generator Parameters",
    "schema": {
        "type": "object",
        "properties": {
            "numSeries": {
                "type": "integer",
                "title": "Number of Series",
                "minimum": 1,
                "maximum": 10,
                "default": 3
            },
            "order": {
                "type": "integer",
                "title": "Order",
                "minimum": 1,
                "maximum": 8,
                "default": 5
            },
            "multiplier": {
                "type": "integer",
                "title": "Multiplier",
                "minimum": 1,
                "maximum": 10,
                "default": 1
            }
        }
    },
    "uiSchema": {
        "numSeries": {"ui:widget": "range"},
        "order": {"ui:widget": "range"},
        "multiplier": {"ui:widget": "range"}
    }
})";

// --- Plugin Logic ---

bool show_host_form() {
  std::string schema_str(formSchema);
  // Remove newlines to keep it a single-line JSON for the IPC protocol
  schema_str.erase(std::remove(schema_str.begin(), schema_str.end(), '\n'),
                   schema_str.end());
  schema_str.erase(std::remove(schema_str.begin(), schema_str.end(), '\r'),
                   schema_str.end());

  sdk::send_response(schema_str);

  // Read response from host (stdin)
  std::string response;
  if (!std::getline(std::cin, response)) {
    return false;
  }

  // Check for error/cancelled
  if (response.find("\"error\"") != std::string::npos) {
    return false;
  }

  // Parse result
  std::string_view series_str = sdk::find_json_value(response, "numSeries");
  std::string_view order_str = sdk::find_json_value(response, "order");
  std::string_view mult_str = sdk::find_json_value(response, "multiplier");

  bool updated = false;
  if (!series_str.empty()) {
    try {
      g_config.numSeries = std::stoi(std::string(series_str));
      updated = true;
    } catch (...) {
    }
  }
  if (!order_str.empty()) {
    try {
      g_config.order = std::stoi(std::string(order_str));
      updated = true;
    } catch (...) {
    }
  }
  if (!mult_str.empty()) {
    try {
      g_config.multiplier = std::stod(std::string(mult_str));
      updated = true;
    } catch (...) {
    }
  }

  if (updated) {
    g_config.numPoints =
        static_cast<int>(g_config.multiplier * std::pow(10, g_config.order));
    sdk::log_info(std::format(
        "Config updated: points={}, series={}, order={}, multiplier={:.2f}",
        g_config.numPoints, g_config.numSeries, g_config.order,
        g_config.multiplier));
  }

  return updated;
}

void generate_data(std::string_view series_id) {
  sdk::log_info(std::format("Generating data for series: {}", series_id));

  // Unique seed per series to ensure different data
  std::hash<std::string_view> hasher;
  unsigned int series_seed = static_cast<unsigned int>(std::time(nullptr)) ^
                             static_cast<unsigned int>(hasher(series_id));

  std::mt19937 gen(series_seed);
  std::normal_distribution<double> dist(0.0, 1.0);
  std::uniform_real_distribution<double> dt_dist(0.1, 10.0);

  std::vector<double> result;
  result.reserve((static_cast<size_t>(g_config.numPoints) + 1) * 2);

  double t = 0;
  double y = 0;
  result.push_back(t);
  result.push_back(y);

  for (int i = 0; i < g_config.numPoints; ++i) {
    double dt = dt_dist(gen);
    t += dt;
    y += dist(gen) * std::sqrt(dt) * g_config.noise;
    result.push_back(t);
    result.push_back(y);
  }

  sdk::send_binary_data(result, "interleaved");
}

int main(int argc, char *argv[]) {
  // Check for --metadata flag
  for (int i = 1; i < argc; ++i) {
    if (std::string_view(argv[i]) == "--metadata") {
      std::cout << R"({"name":"Random Walk Generator","patterns":[]})"
                << std::endl;
      return 0;
    }
  }

  std::string line;
  while (std::getline(std::cin, line)) {
    if (line.empty())
      continue;

    if (line.find("\"method\":\"info\"") != std::string::npos) {
      sdk::send_response(std::format("{{\"name\":\"{}\",\"version\":{}}}",
                                     pluginName, pluginVersion));
    } else if (line.find("\"method\":\"initialize\"") != std::string::npos) {
      bool ok = show_host_form();
      if (ok) {
        sdk::send_response("{\"result\":\"initialized\"}");
      } else {
        sdk::send_response("{\"error\":\"cancelled\"}");
      }
    } else if (line.find("\"method\":\"get_chart_config\"") !=
               std::string::npos) {
      sdk::send_response("{\"result\":{\"title\":\"C++ Random "
                         "Walk\",\"axis_labels\":[\"Time\",\"Value\"]}}");
    } else if (line.find("\"method\":\"get_series_config\"") !=
               std::string::npos) {
      std::string items = "";
      for (int i = 0; i < g_config.numSeries; ++i) {
        if (i > 0)
          items += ",";
        items += std::format("{{\"id\":\"series_{}\",\"name\":\"C++ Series "
                             "{}\"}}",
                             i, i + 1);
      }
      sdk::send_response(std::format("{{\"result\":[{}]}}", items));
    } else if (line.find("\"method\":\"get_series_data\"") !=
               std::string::npos) {
      std::string_view line_view = line;
      std::string_view sid = "series_0";
      size_t pos = line_view.find("\"series_id\":\"");
      if (pos != std::string_view::npos) {
        size_t start = pos + 13;
        size_t end = line_view.find("\"", start);
        if (end != std::string_view::npos) {
          sid = line_view.substr(start, end - start);
        }
      }
      generate_data(sid);
    }
  }
  return 0;
}
