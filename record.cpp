#include "record.h"


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