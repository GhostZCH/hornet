package store

type Item struct {
	Key        Key
	Block      int64
	Offset     int64
	HeaderLen  int64
	BodyLen    int64
	UserGroup  int64
	User       int64
	RootDomain int64
	Domain     int64
	SrcGroup   int64
	Expires    int64
	Path       []byte
	Tags       int64
}
