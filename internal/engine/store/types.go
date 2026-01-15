package store

type ValueType int

const (
	StringType ValueType = iota
	ListType
	SetType
	HashType
)
