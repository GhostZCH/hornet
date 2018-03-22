#include "disk.h"

#include <sstream>
#include <memory>
#include <algorithm>

#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>


Disk::Disk(const string& path, const uint32_t block_count, const uint32_t now)
{
    now_ = now;
    path_ = path;

    block_count_ = block_count;
    
    current_pos_ = 0;
    current_block_ = 0;
}


Disk::~Disk()
{
    for (auto b : blocks_) {
        if (close(b.second.fd) < 0) {
            return;
        }
    }

    int meta_fd = open((path_ + "meta.tmp").c_str(), O_RDWR);
    if (meta_fd < 0) {
        return;
    }

    MetaHeader header;
    header.block_start = -1;
    header.block_end = 0;
    header.item_count = 0;

    for (auto b : blocks_) {
        if (b.second.deleted) {
            continue;
        }

        if (b.first < header.block_start) {
            header.block_start = b.first;
        }
        
        if (b.first > header.block_end) {
            header.block_end = b.first;
        }
    }

    if (lseek(meta_fd, sizeof(header), SEEK_SET) < 0) {
        return;
    }

    DiskItem ditem;
    for (auto dir_map : meta_) {
        for (auto id_item : dir_map.second) {
            if (verify_item(id_item.second, now_)) {
                to_disk_item(ditem, dir_map.first, id_item.first, id_item.second);

                if (write(meta_fd, &ditem, sizeof(ditem)) == sizeof(ditem)){
                    return;
                }

                header.item_count++;
            }
        }
    }

    if (lseek(meta_fd, 0, SEEK_SET) < 0){
       return;
    }

    if (write(meta_fd, &header, sizeof(header)) != sizeof(header)) {
        return;
    }

    if (close(meta_fd) < 0) {
        return;
    }

    rename((path_ + "meta.tmp").c_str(), (path_ + "meta").c_str());
}


bool Disk::Init()
{
    string mate_file = path_ + "meta";

    if (unlink(mate_file.c_str()) < 0) {
        return false;
    }

    int meta_fd = open(mate_file.c_str(), O_RDWR);
    if (meta_fd < 0) {
        return false;
    }

    MetaHeader header;
    if (read(meta_fd, &header, sizeof(header)) < sizeof(header)) {
        return false;
    }

    Block block;
    for (uint32_t i = header.block_start; i <= header.block_end; i++) {
        block.use = 0;
        block.deleted = 0;

        block.fd = open(to_string(i).c_str(), O_RDWR);
        if (block.fd > 0) {
            blocks_[i] = block;
            current_block_ = i;
        }
    }
    current_pos_ = block_size_; // use new block when add

    Item item;
    DiskItem ditem;
    for (uint32_t i = 0; i < header.item_count; i++) {
        if (read(meta_fd, &ditem, sizeof(ditem)) < sizeof(ditem)) {
            return false;
        }

        if (blocks_.find(ditem.block) == blocks_.end()) {
            continue;
        }

        to_item(item, ditem);
        meta_[ditem.dir][ditem.id] = item;
    }

    return close(meta_fd) >= 0;
}


void Disk::UpdateTime(uint32_t now)
{
    now_ = now;
}


Item* Disk::Add(const Key& dir, const Key& id, Item& item)
{
    if (item.size + current_pos_ > block_size_ ) {
        if (!nextBlock()) {
            return nullptr;
        }
    }

    item.block = current_block_;
    item.pos = current_pos_;
    item.putting = 1;
    item.use = 0;

    current_pos_ += item.size;
    meta_[dir][id] = item;
    return &meta_[dir][id];
}


bool Disk::nextBlock()
{
    uint32_t n = 0;
    uint32_t min = (uint32_t)-1;

    for (auto b : blocks_) {
        if (b.second.deleted == 0) {
            if (b.first < min) {
                min = b.first;
            }
            n++;
        }
    }

    if (n > block_count_ && min != (uint32_t)-1) {
        blocks_[min].deleted = 1;
    }
    current_block_++;
    current_pos_ = 0;

    Block block = {-1, 0, 0};
    block.fd = open((path_ + to_string(current_block_)).c_str(), O_RDWR);
    if (block.fd < 0) {
        return false;
    }

    blocks_[current_block_] = block;
    return true;
}