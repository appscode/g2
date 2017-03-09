//using key as queue

package leveldbq

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/appscode/g2/pkg/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDbQ struct {
	db *leveldb.DB
}

var _ storage.Db = &LevelDbQ{}

func New(dir string) (storage.Db, error) {
	db, err := leveldb.OpenFile(strings.TrimRight(dir, "/")+"/gearmand.ldb", nil)
	if err != nil {
		return nil, err
	}
	return &LevelDbQ{db: db}, nil
}

func (q *LevelDbQ) Add(item storage.DbItem) error {
	buf, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return q.db.Put([]byte(item.Key()), buf, nil)
}

func (q *LevelDbQ) Delete(item storage.DbItem) error {
	return q.db.Delete([]byte(item.Key()), nil)
}

func (q *LevelDbQ) Get(t storage.DbItem) error {
	data, err := q.db.Get([]byte(t.Key()), nil)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, t)
	if err != nil {
		return err
	}
	return nil
}

func (q *LevelDbQ) GetAll(t storage.DbItem) ([]storage.DbItem, error) {
	items := make([]storage.DbItem, 0)
	iter := q.db.NewIterator(util.BytesPrefix([]byte(t.Prefix())), nil)
	for iter.Next() {
		obj := reflect.New(reflect.TypeOf(t).Elem()).Interface().(storage.DbItem)
		err := json.Unmarshal(iter.Value(), &obj)
		if err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return items, nil
}
