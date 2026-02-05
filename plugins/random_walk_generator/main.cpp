#ifndef UNICODE
#define UNICODE
#endif

#include <windows.h>
#include <iostream>
#include <string>
#include <string_view>
#include <vector>
#include <sstream>
#include <iomanip>
#include <cmath>
#include <random>
#include <algorithm>
#include <array>
#include <ctime>
#include <fcntl.h>
#include <io.h>
#include <format>

#pragma comment(linker, "/SUBSYSTEM:windows /ENTRY:mainCRTStartup")

// Forward declarations
void send_response(std::string_view json);

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

// Colors for series
constexpr std::array<std::string_view, 10> ChartColors = {
    "#636EFA", "#EF553B", "#00CC96", "#AB63FA", "#FFA15A",
    "#19D3F3", "#FF6692", "#B6E880", "#FF97FF", "#FECB52"
};

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

// --- IPC Communication Helpers ---

void send_response(std::string_view json) {
    std::cout << json << std::endl;
}

void log_message(std::string_view level, std::string_view message) {
    std::cout << std::format("{{\"method\":\"log\",\"level\":\"{}\",\"message\":\"{}\"}}", level, message) << std::endl;
}

void log_info(std::string_view msg) { log_message("info", msg); }
void log_error(std::string_view msg) { log_message("error", msg); }
void log_debug(std::string_view msg) { log_message("debug", msg); }

// Simple helper to find a value in a simple JSON object string
std::string_view find_json_value(std::string_view json, std::string_view key) {
    std::string search_key = "\"";
    search_key += key;
    search_key += "\":";
    
    size_t pos = json.find(search_key);
    if (pos == std::string_view::npos) return "";

    size_t val_start = pos + search_key.length();
    // Skip spaces, colons, and open braces/quotes
    while (val_start < json.length() && (json[val_start] == ' ' || json[val_start] == ':' || json[val_start] == '{')) val_start++;

    if (val_start >= json.length()) return "";

    if (json[val_start] == '"') {
        // String value
        val_start++;
        size_t val_end = json.find('"', val_start);
        if (val_end == std::string_view::npos) return "";
        return json.substr(val_start, val_end - val_start);
    } else {
        // Numeric value
        size_t val_end = val_start;
        while (val_end < json.length() && (isdigit(json[val_end]) || json[val_end] == '.' || json[val_end] == '-')) val_end++;
        return json.substr(val_start, val_end - val_start);
    }
}

// --- Plugin Logic ---

bool show_host_form() {
    std::string schema_str(formSchema);
    // Remove newlines to keep it a single-line JSON for the IPC protocol
    schema_str.erase(std::remove(schema_str.begin(), schema_str.end(), '\n'), schema_str.end());
    schema_str.erase(std::remove(schema_str.begin(), schema_str.end(), '\r'), schema_str.end());

    send_response(schema_str);

    // Read response from host (stdin)
    std::string response;
    if (!std::getline(std::cin, response)) return false;

    // Check for error/cancelled
    if (response.find("\"error\"") != std::string::npos) return false;

    // Parse result
    std::string_view series_str = find_json_value(response, "numSeries");
    std::string_view order_str = find_json_value(response, "order");
    std::string_view mult_str = find_json_value(response, "multiplier");

    bool updated = false;
    if (!series_str.empty()) {
        try {
            g_config.numSeries = std::stoi(std::string(series_str));
            updated = true;
        } catch (...) {}
    }
    if (!order_str.empty()) {
        try {
            g_config.order = std::stoi(std::string(order_str));
            updated = true;
        } catch (...) {}
    }
    if (!mult_str.empty()) {
        try {
            g_config.multiplier = std::stod(std::string(mult_str));
            updated = true;
        } catch (...) {}
    }

    if (updated) {
        g_config.numPoints = static_cast<int>(g_config.multiplier * std::pow(10, g_config.order));
        log_info(std::format("Config updated: points={}, series={}, order={}, multiplier={:.2f}", 
            g_config.numPoints, g_config.numSeries, g_config.order, g_config.multiplier));
    }

    return updated;
}

void generate_data(std::string_view series_id) {
    log_info(std::format("Generating data for series: {}", series_id));
    
    // Unique seed per series to ensure different data
    std::hash<std::string_view> hasher;
    unsigned int series_seed = static_cast<unsigned int>(std::time(nullptr)) ^ static_cast<unsigned int>(hasher(series_id));
    
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

    // Send header
    size_t byte_len = result.size() * sizeof(double);
    std::cout << std::format("{{\"type\":\"binary\",\"length\":{},\"storage\":\"interleaved\"}}", byte_len) << std::endl;
    std::cout.flush();

    // Send binary data (stdout must be in binary mode)
    _setmode(_fileno(stdout), _O_BINARY);
    fwrite(result.data(), 1, byte_len, stdout);
    fflush(stdout);
    _setmode(_fileno(stdout), _O_TEXT);
}

int main(int argc, char* argv[]) {
    // Check for --metadata flag
    for (int i = 1; i < argc; ++i) {
        if (std::string_view(argv[i]) == "--metadata") {
            std::cout << R"({"name":"Random Walk Generator","patterns":[]})" << std::endl;
            return 0;
        }
    }

    std::string line;
    while (std::getline(std::cin, line)) {
        if (line.empty()) continue;

        if (line.find("\"method\":\"info\"") != std::string::npos) {
            send_response(std::format("{{\"name\":\"{}\",\"version\":{}}}", pluginName, pluginVersion));
        }
        else if (line.find("\"method\":\"initialize\"") != std::string::npos) {
            bool ok = show_host_form();
            if (ok) {
                send_response("{\"result\":\"initialized\"}");
            } else {
                send_response("{\"error\":\"cancelled\"}");
            }
        }
        else if (line.find("\"method\":\"get_chart_config\"") != std::string::npos) {
            send_response("{\"result\":{\"title\":\"C++ Random Walk\",\"axis_labels\":[\"Time\",\"Value\"]}}");
        }
        else if (line.find("\"method\":\"get_series_config\"") != std::string::npos) {
            std::string items = "";
            for (int i = 0; i < g_config.numSeries; ++i) {
                if (i > 0) items += ",";
                items += std::format("{{\"id\":\"series_{}\",\"name\":\"C++ Series {}\",\"color\":\"{}\"}}",
                                    i, i + 1, ChartColors[i % ChartColors.size()]);
            }
            send_response(std::format("{{\"result\":[{}]}}", items));
        }
        else if (line.find("\"method\":\"get_series_data\"") != std::string::npos) {
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
