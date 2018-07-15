#pragma once

#include "hornet.h"


struct DiskItem
{
	size_t id;
	size_t dir;

	uint64_t block;
	uint16_t tags[TAG_LIMIT];

	uint32_t pos; // pos in block
	uint32_t size; // item size < block_size < 4G
	uint32_t header_size;

	time_t modifed;
	time_t expired;
};


struct Item
{
	bool putting;

	uint32_t block;
	uint16_t tags[TAG_LIMIT];

	uint32_t pos; // pos in block
	uint32_t size; // item size < block_size < 4G
	uint32_t header_size;

	uint32_t modifed; // 2038 is enough
	uint32_t expired; 
};


typedef unordered_map<size_t, shared_ptr<Item>> ItemMap;
typedef unordered_map<size_t, ItemMap> DirMap;


Item* to_item(const DiskItem &ditem);
bool verify_item(const Item* item);
void to_disk_item(DiskItem &ditem, const size_t dir, const size_t id, const Item& item);
