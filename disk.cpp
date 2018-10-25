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
    string temp = path_ + "meta.tmp";
    string meta = path_ + "meta";

    unlink(temp.c_str());

    // write to tmp first
    int meta_fd = open(temp.c_str(), O_RDWR|O_CREAT, S_IRUSR|S_IWUSR);
    if (meta_fd < 0) {
        return;
    }

    MetaHeader header;
    header.block_start = blocks_.begin()->first;
    header.block_end = current_block_;
    header.item_count = 0;

    if (lseek(meta_fd, sizeof(header), SEEK_SET) < 0) {
        return;
    }

    DiskItem ditem;
    for (auto dir_map : meta_) {
        for (auto id_item : dir_map.second) {
            if (verify_item(id_item.second.get())) {
                to_disk_item(ditem, dir_map.first, id_item.first, *id_item.second);
                if (write(meta_fd, &ditem, sizeof(ditem)) != sizeof(ditem)){
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
    rename(temp.c_str(), meta.c_str());
}


void Disk::Init()
{
    string meta_file = path_ + "meta";

    if (access(meta_file.c_str(), F_OK) == -1) {
        return;
    }

    int meta_fd = open(meta_file.c_str(), O_RDWR);
    if (meta_fd < 0) {
        throw SvrError("can not open meta " + meta_file, __FILE__, __LINE__);
    }

    if (unlink(meta_file.c_str()) < 0) {
        throw SvrError("can not unlink meta " + meta_file, __FILE__, __LINE__);
    }

    MetaHeader header;
    if (read(meta_fd, &header, sizeof(header)) < (ssize_t)sizeof(header)) {
        throw SvrError("read meta header fail " + meta_file, __FILE__, __LINE__);
    }

    for (uint32_t i = header.block_start; i <= header.block_end; i++) {
        string block_name = path_ + to_string(i);

        int fd = open(block_name.c_str(), O_RDWR);
        if (fd > 0) {
            Block* block = new Block(fd, block_name);
            blocks_[i] = shared_ptr<Block>(block);
        }

        current_block_ = i;
    }

    current_pos_ = block_size_; // use new block when start

    for (uint32_t i = 0; i < header.item_count; i++) {
        DiskItem ditem;

        if (read(meta_fd, &ditem, sizeof(ditem)) < (ssize_t)sizeof(ditem)) {
            throw SvrError("read item fail " + meta_file, __FILE__, __LINE__);
        }

        if (blocks_.find(ditem.block) == blocks_.end()) {
            continue;
        }

        meta_[ditem.dir][ditem.id] = shared_ptr<Item>(to_item(ditem));
    }

    if (close(meta_fd) != 0) {
        throw SvrError("close meta fail " + meta_file, __FILE__, __LINE__);
    }
}


void Disk::Add(const size_t dir, const size_t id, shared_ptr<Item>& item, shared_ptr<Block> &block)
{
    unique_lock<mutex> lock(meta_mutex_);

    if (blocks_.size() == 0 || (item->size + current_pos_) > block_size_ ) {
        addBlock();
    }

    item->pos = current_pos_;
    item->block = current_block_;
    current_pos_ += item->size;

    block = blocks_[item->block];
    meta_[dir][id] = item;
}


void Disk::Get(const size_t dir, const size_t id, shared_ptr<Item>& item, shared_ptr<Block> &block)
{
    unique_lock<mutex> lock(meta_mutex_);

    if (meta_.find(dir) == meta_.end() || meta_[dir].find(id) == meta_[dir].end()) {
        return;
    }

    item = meta_[dir][id];
    block = blocks_[item->block];
}


uint32_t Disk::Delete(const size_t dir, const size_t id, const uint16_t tags[])
{
    uint32_t deleted = 0;
    unique_lock<mutex> lock(meta_mutex_);

    if (dir == 0) {
        for (auto &dir: meta_) {
            deleted += dir.second.size();
        }
        meta_.clear();
        return deleted;
    }

    if (meta_.find(dir) == meta_.end()) {
        return 0;
    }

    if (id != 0) {
        return (uint32_t)meta_[dir].erase(id);
    }

    auto dirmap = &meta_[dir];
    auto iter = dirmap->begin();

    while(iter != dirmap->end()) {
        bool match = true;

        for (int i = 0; i < TAG_LIMIT; i++) {
            if (tags[i] != uint16_t(-1) && tags[i] != iter->second->tags[i]) {
                match = false;
                break;
            }
        }

        if (match) {
            deleted ++;
            iter = meta_[dir].erase(iter);
        } else {
            iter ++;
        }
    }

    return deleted;
}


void Disk::addBlock()
{
    current_pos_ = 0;

    if (blocks_.size() == block_count_) {
        uint32_t block = blocks_.begin()->first;

        for (auto &dir: meta_) {
            auto j = dir.second.begin();
            while (j != dir.second.end()) {
                if (j->second->block == block) {
                    j = dir.second.erase(j);
                } else {
                    j++;
                }
            }
        }

        blocks_[block]->Delete();
        blocks_.erase(block);
    }

    if (blocks_.size() != 0) {
        current_block_++;
    }

    string name = path_ + to_string(current_block_);

    int fd = open(name.c_str(), O_RDWR|O_CREAT, S_IRUSR|S_IWUSR);
    if (fd < 0) {
        throw SvrError("open block file orror", __FILE__, __LINE__);
    }

    blocks_[current_block_] = shared_ptr<Block>(new Block(fd, name));
}
