#include "disk.h"
#include <sys/sendfile.h>


Disk::Disk(const string& path, const uint32_t block_count, const size_t block_size)
{
    path_ = path;

    block_count_ = block_count;
    block_size_ = block_size;

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

    // write to tmp first
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

    // if everything ok, mv meta.tmp to tmp
    rename((path_ + "meta.tmp").c_str(), (path_ + "meta").c_str());
}


bool Disk::Init()
{
    string meta_file = path_ + "meta";

    if (access(meta_file.c_str(), F_OK) == -1) {
        return nextBlock();
    }

    if (unlink(meta_file.c_str()) < 0) {
        return false;
    }

    int meta_fd = open(meta_file.c_str(), O_RDWR);
    if (meta_fd < 0) {
        return false;
    }

    MetaHeader header;
    if (read(meta_fd, &header, sizeof(header)) < (ssize_t)sizeof(header)) {
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

    for (uint32_t i = 0; i < header.item_count; i++) {
        Item item;
        DiskItem ditem;

        if (read(meta_fd, &ditem, sizeof(ditem)) < (ssize_t)sizeof(ditem)) {
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

    current_pos_ += item.size;
    meta_[dir][id] = item;
    return &meta_[dir][id];
}


Item* Disk::Get(const Key& dir, const Key& id)
{
    if (meta_.find(dir) == meta_.end()) {
        return nullptr;
    }

    if (meta_[dir].find(id) == meta_[dir].end()) {
        return nullptr;
    }

    Item* item = &meta_[dir][id];
    if (!verify_item(*item, now_)) {
        return nullptr;
    }

    return item;
}


uint32_t Disk::Delete(const Key &dir, const Key &id, const uint16_t tags[])
{
    if (meta_.find(dir) == meta_.end()) {
        return 0;
    }

    if (id == NULL_ITEM_KEY) {
        uint32_t deleted = 0;

        for (auto id_item: meta_[dir]) {
            id_item.second.deleted = 1;

            for (int i=0; i < TAG_LIMIT; i++) {
                if (tags[i] != uint16_t(-1)
                   && tags[i] != id_item.second.tags[i]) {
                    id_item.second.deleted = 0;
                    break;
                }
            }

            deleted += id_item.second.deleted;
        }

        return deleted;
    }

    if (meta_[dir].find(id) == meta_[dir].end()) {
        return false;
    }

    meta_[dir][id].deleted = 1;
    return 1;
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
    block.fd = open((path_ + to_string(current_block_)).c_str(), O_RDWR|O_CREAT, S_IRUSR|S_IWUSR);
    if (block.fd < 0) {
        return false;
    }

    blocks_[current_block_] = block;
    return true;
}


ssize_t Disk::Wirte(Item *item, char* buf)
{
    return pwrite(blocks_[item->block].fd, buf, item->size, item->pos);
}


ssize_t Disk::Read(Item *item, char* buf)
{
    return pread(blocks_[item->block].fd, buf, item->size, item->pos);
}


ssize_t Disk::Send(Item *item, int sock, off_t off)
{
    off_t start = off + item->pos;
    return sendfile(sock, blocks_[item->block].fd, &start, item->size - off);
}
