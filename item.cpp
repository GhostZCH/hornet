#include "item.h"
#include "tool.h"


Item* to_item(const DiskItem &ditem)
{
	Item* item = new Item();

	item->block = ditem.block;
	item->pos = ditem.pos;
	item->size = ditem.size;
	item->header_size = ditem.header_size;
	item->modifed = ditem.modifed;
	item->expired = ditem.expired;

	for (int i = 0; i < TAG_LIMIT; i++) {
		item->tags[i] = ditem.tags[i];
	}

	return item;
}


void to_disk_item(DiskItem &ditem, const size_t dir, const size_t id, const Item& item)
{
	bzero(&ditem, sizeof(item));

	ditem.dir = dir;
	ditem.id = id;
	ditem.block = item.block;
	ditem.pos = item.pos;
	ditem.size = item.size;
	ditem.header_size = item.header_size;
	ditem.modifed = item.modifed;
	ditem.expired = item.expired;

	for (int i = 0; i < TAG_LIMIT; i++) {
		ditem.tags[i] = item.tags[i];
	}
}


bool verify_item(const Item* item)
{
	return !item->putting && item->expired > g_hornet_now;
}
