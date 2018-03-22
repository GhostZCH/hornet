#pragma once

#include "hornet.h"


template<typename T>
class Slab
{
public:
	Slab();
	~Slab();
	bool Init(size_t limit);
	T* Get();
	bool Free(T* item);

private:
	T* data_;
    size_t limit_;
	unordered_set<T*> free_;
};


template<typename T>
Slab<T>::Slab()
{
	data_ = NULL;
	limit_ = 0;
}


template<typename T>
Slab<T>::~Slab()
{
	free(data_);
}


template<typename T>
bool Slab<T>::Init(size_t limit)
{
	limit_ = limit;
	data_ = new T[limit];
	if (data_ == nulptr) {
		return false;
	}

	for (size_t i = 0; i < limit; i++) {
		free_.insert(data_ + i);
	}

	return true;
}


template<typename T>
T * Slab<T>::Get()
{
	if (free_.begin() == free_.end()) {
		return nullptr;
	}

	T* item = *(free_.begin());
	free_.erase(item);

	return item;
}


template<typename T>
bool Slab<T>::Free(T * item)
{
	if (pos >= limit_) {
		return false;
	}

	free_.insert(item);
    return true;
}
