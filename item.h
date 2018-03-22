#pragma once

#include "hornet.h"


struct Key
{
	uint64_t data[2];
	bool operator == (const Key & other) const;
	void Load(const char *in);
	char* Dump(char *out);
};


const int KEY_CHAR_SIZE = sizeof(Key) * 2;
const Key NULL_KEY = {{-1,-1}};


struct KeyHash
{
	size_t operator ()(const Key& k) const;
};


struct KeyEqual
{
	bool operator () (const Key &k1, const Key &k2) const;
};


struct DiskItem
{
	Key id;
	Key dir;

	uint64_t block;
	uint16_t tags[TAG_LIMIT];

	uint32_t pos; // pos in block
	uint32_t size; // item size < block_size < 4G
	uint32_t header_size;

	time_t modifed;
	time_t expired;

	char etag[ETAG_LIMIT];
};


struct Item
{
	uint32_t use:30; // 2 ^ 30 is enough
	uint32_t deleted:1;
	uint32_t putting:1;

	uint32_t block;
	uint16_t tags[TAG_LIMIT];

	uint32_t pos; // pos in block
	uint32_t size; // item size < block_size < 4G
	uint32_t header_size;

	uint32_t modifed; // 2038 is enough
	uint32_t expired; 

	char etag[ETAG_LIMIT];
};


typedef unordered_map<Key, Item, KeyHash, KeyEqual> ItemMap;
typedef unordered_map<Key, ItemMap, KeyHash, KeyEqual> DirMap; 

bool verify_item(const Item& item, uint32_t now);
void to_item(Item& item, const DiskItem &ditem);
void to_disk_item(DiskItem &ditem, const Key& dir, const Key& id, const Item& item);
