package storage

type Db interface {
	ItemQueue
}

type DbItem interface {
	Key() string
	Prefix() string
}

type ItemQueue interface {
	Add(item DbItem) error
	Delete(item DbItem) error
	Get(t DbItem) error
	GetAll(t DbItem) ([]DbItem, error)
}
