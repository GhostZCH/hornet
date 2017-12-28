#pragma once


#include "hornet.h"


typedef unordered_map<Key, Record, KeyHash, KeyEqual> RecordMap; 
typedef bool (*RecordJudge)(const Record& r);


struct MetaStat
{
    off_t current_block;
    off_t current_pos;
};


class Device 
{
public:
    const size_t BLOCK_SIZE = 512 * 1024 * 1024;

public:
    Device(const string& meta_path, string& data_path);
    ~Device();

    bool Init();

    bool LoadMeta();
    bool DumpMeta();
    size_t Size();

    size_t Delete(const Key& k);
    bool Add(Record& r, char *content);
    Record* Get(const Key& k);
    size_t DeleteByDir(const Key& dir);
    size_t DeleteByBlock(const off_t block);

private:
    bool OpenFile();
    bool Write(const char *content, Record &record);
    size_t DeleteBatch(function<bool(const Record&)> judge);

private:
    short id_;    
    string meta_dir_;
    string data_dir_;

    size_t block_count_;
    size_t capacity_;

    MetaStat stat_;
    RecordMap map_;

    int meta_fd_;
    int data_fd_;
};