#pragma once

#include "hornet.h"


const int LOG_INFO = 1;
const int LOG_WARN = 2;
const int LOG_ERROR = 3;

extern time_t g_hornet_now;
extern time_t g_hornet_now_ms;

extern map<string, string> g_config;

extern ostream* g_logger;
extern int g_loglevel;
extern const char *g_log_level_str[];

void update_time();
string get_time_str();

bool set_logger(const string& level, ostream *logger);

#define logger(level, stream) do { \
    if ((level) >= g_loglevel) { \
        (*g_logger) << get_time_str() << " [" << g_log_level_str[level] << "] "<< stream << endl;} \
    } while(0)

bool load_conf(string filename);
bool get_param(int argc, char *argv[], map<string, pair<string, string>>& params);
