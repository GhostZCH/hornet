#include "access_log.h"
#include "tool.h"

AccessLog::AccessLog(string &path)
{
    fd_ = -1;
    need_reopen_ = false;
    path_ = path;
    buffer_ = unique_ptr<char []>(new char[ACCESS_LOG_BUF]);
}


bool AccessLog::Init()
{
    return openFile();
}


void AccessLog::Reopen()
{
    // async
    need_reopen_ = true;
}


bool AccessLog::Log(char* buf, ssize_t n)
{
    if(n <= 0 || write(fd_, buf, (size_t)n) != n) {
        LOG(LERROR, "write access log failed: errno=" << errno << " fd=" << fd_);
        return false;
    }

    if (need_reopen_) {
        need_reopen_ = false;
        close(fd_);
        return openFile();
    }

    return true;
}


bool AccessLog::openFile()
{
    fd_ = open(path_.c_str(), O_WRONLY|O_APPEND|O_CREAT, S_IWUSR|S_IRUSR);

    if (fd_ <= 0) {
        LOG(LERROR, "open access file [" << get_conf("log.access") << "] error");
        return false;
    }

    return true;
}