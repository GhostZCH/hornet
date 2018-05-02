#pragma once

#include "hornet.h"

using namespace std;

class Pool
{
public:
    Pool(size_t limit, size_t page=4096);
    ~Pool();

    char* Alloc(size_t size);
    size_t Size();
private:
    bool NewPage();

    size_t limit_;
    size_t size_;
    size_t page_;

    size_t current_page_;
    size_t current_pos_;

    vector<char *> pages_;
    vector<char *> big_;
};