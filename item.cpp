#include "item.h"


void Key::Load(const char *in) 
{
	char *cdata = (char *)&data;
	data[0] = data[1] = 0;

	// ERROR
	for (int i = 0; i < KEY_CHAR_SIZE; i++) {
		char c = in[i] >= '0' && in[i] <= '9' ? in[i] - '0' : in[i] - 'a' + 10; 
		cdata[i/2] += c << (i & 1 ? 0 : 4); // (i + 1) & 1 == (i + 1) % 2 
	}
}


char* Key::Dump(char *out)
{
	char *cdata = (char *)&data;
	char map[] = "0123456789abcdef";

	for (int i = 0; i < KEY_CHAR_SIZE / 2; i++) {
		out[i * 2] = map[(cdata[i] & 0xF0) >> 4];
		out[i * 2 + 1] = map[cdata[i] & 0xF];
	}

	return out;
}


bool Key::operator==(const Key & other) const
{
	return data[0] == other.data[0] && data[1] == other.data[1];
}


bool KeyEqual::operator()(const Key & k1, const Key & k2) const
{
	return k1 == k2;
}


size_t KeyHash::operator()(const Key & k) const
{
	return (size_t)k.data[0];
}


void to_item(Item& item, const DiskItem &ditem)
{
	bzero(&item, sizeof(item));
	item.block = ditem.block;
	item.pos = ditem.pos;
	item.size = ditem.size;
	item.header_size = ditem.header_size;
	item.modifed = ditem.modifed;
	item.expired = ditem.expired;

	for (int i = 0; i < TAG_LIMIT; i++) {
		item.tags[i] = ditem.tags[i];
	}

	memcpy(item.etag, ditem.etag, ETAG_LIMIT);
}


void to_disk_item(DiskItem &ditem, const Key& dir, const Key& id, const Item& item)
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

	memcpy(ditem.etag, item.etag, ETAG_LIMIT);
}


bool verify_item(const Item& item, uint32_t now)
{
	return (item.putting != 1 && item.deleted != 1 && item.expired > now);
}
