package store

type Item struct {
	Key        Key
	Block      int64
	Offset     int64
	HeaderLen  int64
	BodyLen    int64
	UserGroup  uint64
	User       uint64
	RootDomain uint64
	Domain     uint64
	SrcGroup   uint64
	Expires    int64
	Path       []byte
	Tags       int64
}
