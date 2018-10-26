#include "access_log.h"
#include "tool.h"

AccessLog::AccessLog(string &path)
{
    fd_ = -1;
    need_reopen_ = false;
    path_ = path;
    buffer_ = unique_ptr<char []>(new char[ACCESS_LOG_BUF]);
}


void AccessLog::Init()
{
    fd_ = open(path_.c_str(), O_WRONLY|O_APPEND|O_CREAT, S_IWUSR|S_IRUSR);
    if (fd_ <= 0) {
        throw SvrError("open access log failed", __FILE__, __LINE__);
    }
}


void AccessLog::Reopen()
{
    need_reopen_ = true;
}


void AccessLog::Log(char* buf, ssize_t n)
{
    if(n <= 0 || write(fd_, buf, (size_t)n) != n) {
        throw SvrError("write access failed " + string(buf) + " " + to_string(n), __FILE__, __LINE__);
    }

    if (need_reopen_) {
        need_reopen_ = false;
        close(fd_);
        fd_ = -1;
        Init();
    }
}
