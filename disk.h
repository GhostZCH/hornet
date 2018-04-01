#pragma once

#include "hornet.h"

struct Block
{
    int fd;
    uint32_t use:31;
    uint32_t deleted:1;
};


struct MetaHeader
{
    uint32_t magic;
    uint32_t block_start;
    uint32_t block_end;
    uint32_t item_count;
};


typedef map<uint32_t, Block> BlockMap;


class Disk
{
public:
    Disk(const string& path, const uint32_t block_count, const uint32_t now);
    ~Disk();

    bool Init();
    void UpdateTime(uint32_t now);

    Item* Add(const Key& dir, const Key& id, Item& item);
    Item* Get(const Key& dir, const Key& id);
    uint32_t Delete(const Key &dir, const Key &id, const uint16_t tags[]);

private:
    bool nextBlock();

private:
    string path_;

    uint32_t now_;

    uint32_t block_count_;
    uint32_t block_size_;

    uint32_t current_block_; 
    uint32_t current_pos_;

    DirMap meta_;
    BlockMap blocks_;
};
// TODO block status use and deleted
