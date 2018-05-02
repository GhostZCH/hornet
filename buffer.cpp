#include "buffer.h"


Buffer::Buffer(char* start, size_t limit)
{
    start_ = start;
    end_ = start + limit;
}

bool Buffer::Read();
bool Buffer::Write();
bool Buffer::Finish();

