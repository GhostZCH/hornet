#pragma once

#include "hornet.h"
#include "tool.h"
#include "item.h"

class Mem
{
public:
    Mem(size_t size);

private:

    DirMap meta_;
    map<uint32_t, shared_ptr<Block>> blocks_;

    mutex meta_mutex_;
};
