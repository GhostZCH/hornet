#pragma once

#include "hornet.h"


const int LDEBUG = 1;
const int LWARN = 2;
const int LERROR = 3;

extern time_t g_now;
extern time_t g_now_ms;


extern ostream* g_logger;
extern int g_loglevel;
extern const char *g_log_level_str[];

void update_time();
string get_time_str();

bool set_logger(const string& level, ostream *logger);

#define LOG(level, stream) do { \
    if ((level) >= g_loglevel) { \
        (*g_logger) << get_time_str() << " [" << g_log_level_str[level] << "] "<< stream << endl;} \
    } while(0)

void load_conf(string filename);
string& get_conf(const string& name);
void get_param(int argc, char *argv[], map<string, pair<string, string>>& params);


class SvrError:public exception
{
public:
    SvrError(const string& msg, const char* file, int line) {
        stringstream ss;
        ss <<  file << "[" << line << "]: " << msg
           << "(" << (strerror(errno) == nullptr ? "NULL" : strerror(errno)) << ")";
        msg_ = ss.str();
    }

    const char* what() const noexcept {return msg_.c_str();}

private:
    string msg_;
};


class ReqError
{
public:
    ReqError(const string& msg):msg_(msg){}
    string& Msg(){return msg_;}

private:
    string msg_;
};
