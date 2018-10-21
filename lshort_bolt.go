package lshort

import (
	"strconv"

	"github.com/boltdb/bolt"
	base58 "github.com/itchyny/base58-go"
	"github.com/pkg/errors"
)

const BoltBucketKey = "_links"

type linkShorterBolt struct {
	db *bolt.DB
}

var _ LinkShortner = (*linkShorterBolt)(nil)

func (l *linkShorterBolt) Shrink(url string) (string, error) {
	var key []byte
	err := l.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BoltBucketKey))

		i, err := b.NextSequence()
		if err != nil {
			return errors.Wrap(err, "unable to get next sequence")
		}

		key, err = base58.RippleEncoding.Encode([]byte(strconv.FormatUint(i, 10)))
		if err != nil {
			return errors.Wrap(err, "unable to encode to base58")
		}
		return b.Put(key, []byte(url))
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to add key to database")
	}

	return string(key), nil
}
func (l *linkShorterBolt) Expand(key string) (string, error) {
	var url []byte
	err := l.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BoltBucketKey))
		url = b.Get([]byte(key))
		return nil
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get key from database")
	}
	return string(url), nil
}

func (l *linkShorterBolt) Close() error {
	return l.db.Close()
}

func NewLinkShortnerBolt(dbPath string, opts *bolt.Options) (LinkShortner, error) {
	if opts == nil {
		opts = bolt.DefaultOptions
	}
	db, err := bolt.Open(dbPath, 0666, opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open database")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BoltBucketKey))
		return err
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create bucket in database")
	}
	return &linkShorterBolt{db: db}, nil
}
