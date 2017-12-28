#pragma once

#include "hornet.h"


struct Key
{
	size_t data[2];
	bool operator == (const Key & other) const;
	void Load(const char *in);
	char* Dump(char *out);
};

const int KEY_CHAR_SIZE = sizeof(Key) * 2;

struct KeyHash
{
	size_t operator ()(const Key& k) const;
};


struct KeyEqual
{
	bool operator () (const Key &k1, const Key &k2) const;
};

const Key NULL_KEY = {
	numeric_limits<size_t>::max(),
	numeric_limits<size_t>::max()
};


struct Record
{
	Key id;
	Key dir;
	Key check_sum;
	Key etag;
	time_t expired;
	time_t modifed;
	off_t block;
	off_t start;
	size_t length;
};



