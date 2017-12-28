#include "device.h"

#include <sstream>
#include <memory>
#include <algorithm>

#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>

const size_t BUF_COUNT = 1024 * 1024;
const size_t BUF_SIZE = BUF_COUNT * sizeof(Record);


Device::Device(const string& meta_dir, string& data_dir, const short id, const size_t capacity) 
{
    id_ = id;
    capacity_ = capacity;
    block_count_ = capacity / BLOCK_SIZE;

    meta_dir_ = meta_dir;
    data_dir_ = data_dir;

    meta_fd_ = -1;
    data_fd_ = -1;
}


Device::~Device() 
{
    DumpMeta();
    close(meta_fd_);
    close(data_fd_);
}


bool Device::Init() 
{
    if (!OpenFile()) {
        return false;
    }

    if (!LoadMeta()) {
        return false;
    }

    return true;
}


bool Device::LoadMeta()
{
    lseek(meta_fd_, 0, SEEK_SET);

    ssize_t n = read(meta_fd_, &stat_, sizeof(stat_));
    if (n != sizeof(stat_)) {
        return false;
    }

    unique_ptr<Record []> records(new Record[BUF_COUNT]);

    do {
        n = read(meta_fd_, records.get(), BUF_SIZE);
        if (n < 0 || n % sizeof(Record) != 0) {
            return false;
        }

        for (int i = 0; i < n / ssize_t(sizeof(Record)); i++) {
            Record *r = records.get() + i;
            map_[r->id] = *r;
        }
    } while(n == BUF_SIZE);

    return true;
}


bool Device::DumpMeta() 
{
    lseek(meta_fd_, 0, SEEK_SET);
    ftruncate(meta_fd_, 0);

    ssize_t n = write(meta_fd_, &stat_, sizeof(stat_));
    if (n != sizeof(stat_)) {
        return false;
    }

    int index = 0;    
    unique_ptr<Record []> records(new Record[BUF_COUNT]);
    
    for (auto r: map_) {
        records.get()[index++] = r.second;
        if (index != BUF_COUNT) {
            continue;
        }

        n = write(meta_fd_, records.get(), BUF_SIZE);
        if (n != BUF_SIZE) {
            return false;
        }
        index = 0;
    }
    n = write(meta_fd_, records.get(), index * sizeof(Record));
    return n != ssize_t(index * sizeof(Record));
}


size_t Device::Size() 
{
    return map_.size();
}


size_t Device::Delete(const Key& k) 
{
    return map_.erase(k);
}


bool Device::Add(Record& r, char *content) 
{
    if (map_.find(r.id) != map_.end()) {
        return false;
    }

    if (!Write(content, r)) {
        return false;
    }

    map_[r.id] = r;
    return true;
}


Record* Device::Get(const Key& k) 
{
    auto iter  = map_.find(k);
    if (iter == map_.end()) {
        return nullptr;
    }
    return &(iter->second);
}


size_t Device::DeleteByDir(const Key& dir) 
{
    return DeleteBatch(
        [dir](const Record &r)->bool{return r.dir == dir;}
    );
}


size_t Device::DeleteByBlock(const off_t block) {
    return DeleteBatch(
        [block](const Record &r)->bool{return r.block == block;}
    );
}
    

size_t Device::DeleteBatch(function<bool(const Record&)> judge) 
{
    size_t count = 0;
    for (auto iter = map_.begin(); iter != map_.end();) {
        if (judge(iter->second)) {
            count ++;
            iter = map_.erase(iter);
        } else {
            iter ++;
        }
    }
    return count;
}


bool Device::OpenFile() {
    string filename;

    filename = meta_dir_ + to_string(id_) + ".meta";
    meta_fd_ = open(filename.c_str(), O_RDWR);

    filename = data_dir_ + to_string(id_) + ".data";
    data_fd_ = open(filename.c_str(), O_RDWR);

    return data_fd_ > 0 && meta_fd_ >= 0;
}


bool Device::Write(const char *content, Record &record) 
{
    if (stat_.current_pos + record.length > BLOCK_SIZE) {
        DeleteByBlock(stat_.current_block);
        stat_.current_pos = 0;
        stat_.current_block = (stat_.current_block + 1) % block_count_;
        DumpMeta();
    }

    off_t offset = BLOCK_SIZE * stat_.current_block + stat_.current_pos;    

    size_t n = pwrite(data_fd_, content, record.length, offset);
    if (n != record.length) {
        return false;
    }

    stat_.current_pos += record.length;

    return true;
}
