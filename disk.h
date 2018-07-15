#pragma once

#include "hornet.h"
#include "tool.h"
#include "item.h"

class Block
{
public:
    Block(int fd, string& name):fd_(fd),name_(name){};
    ~Block(){close(fd_);unlink(name_.c_str());}
    bool Wirte(Item *item, const char* buf, uint32_t len, off_t off);
    bool Read(Item *item, char* buf);
    bool Send(Item *item, int sock, uint32_t& off);

private:
    int fd_;
    string name_;
};


struct MetaHeader
{
    uint32_t magic;
    uint32_t block_start;
    uint32_t block_end;
    uint32_t item_count;
};


class Disk
{
public:
    Disk(const string& path, const uint32_t block_count, const size_t block_size);
    ~Disk();

    bool Init();

    bool Add(const size_t dir, const size_t id, shared_ptr<Item>& item, shared_ptr<Block> &block);
    bool Get(const size_t dir, const size_t id, shared_ptr<Item>& item, shared_ptr<Block> &block);
    uint32_t Delete(const size_t dir, const size_t id, const uint16_t tags[]);

private:
    bool addBlock();

private:
    string path_;

    uint32_t block_count_;
    uint32_t block_size_;

    uint32_t current_block_{0}; 
    uint32_t current_pos_;

    DirMap meta_;
    map<uint32_t, shared_ptr<Block>> blocks_;

    mutex meta_mutex_;
};
