#pragma once
#include "hornet.h"

class Buffer
{
public:
    virtual bool Recv(int sock);
    virtual bool Send(int sock);

    size_t size;
    size_t recved;
    size_t sended;

    // for outer use
    size_t processed;
};

class MemBuffer:public Buffer
{
public:
    MemBuffer(size_t buf_size);

    char *Get(){return data_.get();}
    bool Recv(int sock);
    bool Send(int sock);

private:
    unique_ptr<char []> data_;
}

class FileBuffer:public Buffer
{
public:
    FileBuffer(int fd, off_t off, size_t buf_size);
    bool Recv(int sock);
    bool Send(int sock);

private:
    int fd_;
    off_t off_;
    unique_ptr<char []> tmp_;
}
