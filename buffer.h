#pragma once
#include "hornet.h"
#include "tool.h"
#include <sys/sendfile.h>

class Buffer
{
public:
    virtual void Recv(int sock) = 0;
    virtual void Send(int sock) = 0;

    ssize_t size;
    ssize_t recved;
    ssize_t sended;
    ssize_t processed; // for outer use
};


class MemBuffer:public Buffer
{
public:
    MemBuffer(size_t buf_size);

    char *Get(){return data_.get();}
    void Recv(int sock);
    void Send(int sock);

private:
    unique_ptr<char []> data_;
};


class FileBuffer:public Buffer
{
public:
    FileBuffer(int fd, off_t off, ssize_t buf_size);
    void Recv(int sock);
    void Send(int sock);
    void Write(const char *buf, ssize_t size);

private:
    int fd_;
    off_t off_;
    unique_ptr<char []> tmp_;
};
