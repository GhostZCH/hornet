#include "pool.h"


Pool::Pool(size_t limit, size_t page=4096)
{
    limit_ = limit;
    page_ = page;
    size_ = 0;
    current_page_ = 0;
    current_pos_ = 0; 
}


Pool::~Pool()
{
    for (auto p : pages_) {
        free(p);
    }

    for (auto p: big_) {
        free(p);
    }
}


size_t Pool::Size()
{
    return size_;
}


bool Pool::NewPage()
{
    char *p = (char *)malloc(page_);
    if (p == nullptr) {
        return false;
    }

    pages_.push_back(p);
}


char* Pool::Alloc(size_t size)
{
    char *result;

    if (size + size_ > limit_) {
        return nullptr;
    }

    if (size > page_) {
        result = (char *)malloc(size);

        if (result == nullptr) {
            return nullptr;
        }

        big_.push_back(result);
        size_ += size;

        return result;
    }

    if (pages_.size() == 0 || current_pos_ + size > page_ ) {
        if (!NewPage()) {
            return nullptr;
        }

        current_pos_ = 0;
        current_page_ +=1;
    }

    result = pages_[current_page_] + current_pos_;
    current_pos_ += size;

    return result;
}
