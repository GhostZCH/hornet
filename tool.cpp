#include "tool.h"

using namespace chrono;

ostream *g_logger;
int g_loglevel;

time_t g_hornet_now;
time_t g_hornet_now_ms;

map<string, string> g_config;
const char *g_log_level_str[] = {nullptr, "info", "warn", "error"};

void update_time()
{
	g_hornet_now_ms = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
    g_hornet_now =  g_hornet_now_ms/ 1000;
}

string get_time_str()
{
	char tmp[128] = {0};
    strftime(tmp, sizeof(tmp), "%Y-%m-%d %H:%M:%S", localtime(&g_hornet_now));
	return tmp;
}


bool set_logger(const string& level, ostream *logger)
{
    map<string, int> loglevel;

    loglevel["INFO"] = LOG_INFO;
    loglevel["WARN"] = LOG_INFO;
    loglevel["ERROR"] = LOG_INFO;

    if (loglevel.find(level) == loglevel.end()) {
        return false;
    }

    g_logger = logger;
    g_loglevel = loglevel[level];
    return true;
}


void print_help(const map<string, pair<string, string>>& params)
{
    cout << "hornet [-key1 value1] [-key2 value2] ..." << endl;
    cout << "param key as below:" << endl;

	for (auto i: params) {
		cout << "\t-" << i.first << ":\t" << i.second.first << endl;
	}
}


bool get_param(int argc, char *argv[], map<string, pair<string, string>>& params)
{
	string key;

	for (int i = 1; i < argc; i++) {
		if (argv[i][0] == '-' && key.size() == 0) {
			key = argv[i][1];
            if (params.find(key) == params.end()) {
                print_help(params);
				return false;
			}

		} else if (key.size() > 0) {
			params[key].second = argv[i];
            key.clear();

		} else {
            print_help(params);
            return false;
        }
	}

	return true;
}


bool load_conf(string filename)
{
	string line;
	smatch match;
    regex patten("^([^:]+): *([^# ]+)");

	ifstream conf_file(filename);
	if (!conf_file.is_open()) {
		return false;
	}

	while (getline(conf_file, line)) {
		if (regex_match(line, match, patten) && match.size() == 3) {
			g_config[match[1]] = match[2];
		}
	}

	return true;
}