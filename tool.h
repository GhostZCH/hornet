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


class Error {
public:
    Error(const string& msg, const char* file, int line) {
        msg_ = msg;
        file_ = file;
        line_ = line;
    }

    friend ostream& operator << (ostream& out,const Error& err) {
        out << err.file_ << "[" << err.line_ << "]: " << err.msg_;
        return out;
    }

private:
    int line_;
    string msg_;
    const char* file_;
};


class SvrError: public Error
{
public:
    SvrError(const string& msg, const char* file, int line)
        :Error(msg, file, line){}
};


class ReqError: public Error
{
public:
    ReqError(const string& msg, const char* file, int line)
        :Error(msg, file, line){}
};
