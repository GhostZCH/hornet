#include "tool.h"

using namespace chrono;

ostream *g_logger;
int g_loglevel;

time_t g_now;
time_t g_now_ms;

map<string, string> g_config;
const char *g_log_level_str[] = {nullptr, "debug", "warn", "error"};


void update_time()
{
    g_now_ms = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
    g_now =  g_now_ms / 1000;
}


string get_time_str()
{
    char tmp[128] = {0};
    strftime(tmp, sizeof(tmp), "%Y-%m-%d %H:%M:%S", localtime(&g_now));
    return tmp;
}


bool set_logger(const string& level, ostream *logger)
{
    map<string, int> loglevel;

    loglevel["DEBUG"] = LDEBUG;
    loglevel["WARN"] = LWARN;
    loglevel["ERROR"] = LERROR;

    if (loglevel.find(level) == loglevel.end()) {
        return false;
    }

    g_logger = logger;
    g_loglevel = loglevel[level];
    return true;
}


void print_help(const map<string, pair<string, string>>& params)
{
    cout << "   Welcome to use hornet[" << VERSION_STR << "]\n"<< endl;
    cout << "   commond: hornet [-param1 value1] [-param2 value2] ...\n" << endl;
    cout << "   params:" << endl;

    for (auto i: params) {
        cout << "       -" << i.first << ":\t" << i.second.first << endl;
    }

    cout << "\n" << endl;
}


void get_param(int argc, char *argv[], map<string, pair<string, string>>& params)
{
    string key;

    for (int i = 1; i < argc; i++) {
        if (argv[i][0] == '-' && key.size() == 0) {
            key = argv[i][1];
            if (params.find(key) == params.end()) {
                print_help(params);
                throw SvrError("param error", __FILE__, __LINE__);
            }

        } else if (key.size() > 0) {
            params[key].second = argv[i];
            key.clear();

        } else {
            print_help(params);
            throw SvrError("param error", __FILE__, __LINE__);
        }
    }
}


void load_conf(string filename)
{
    string line;
    smatch match;
    regex patten("^([^:]+): *([^# ]+)");

    ifstream conf_file(filename);
    if (!conf_file.is_open()) {
        throw SvrError("open conf file failed", __FILE__, __LINE__);
    }

    while (getline(conf_file, line)) {
        if (regex_match(line, match, patten) && match.size() == 3) {
            g_config[match[1]] = match[2];
        }
    }
}


string& get_conf(const string& name)
{
    // return "" if not exist
    return g_config[name];
}
