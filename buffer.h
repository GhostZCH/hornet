#pragma once

#include "hornet.h"

class Buffer
{
public:
    Buffer(char* start, size_t limit);

    bool Read(int fd);
    bool Write(int fd);
    bool Finish();

private:
    char* start_;
    char* pos_;
    char* end_;
}