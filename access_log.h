#pragma once

#include "hornet.h"


class AccessLog
{
public:
    AccessLog(string& path);
    ~AccessLog(){if(fd_ > 0){close(fd_);}};

    bool Init();
    bool Log(char* buf, ssize_t n);
    void Reopen();

    char* Buffer(){return buffer_.get();};

private:
    bool openFile();

    int fd_;
    bool need_reopen_;
    string path_;
    unique_ptr<char []> buffer_;
};