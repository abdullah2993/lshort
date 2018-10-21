package lshort

import (
	"strconv"

	"github.com/dgraph-io/badger"
	base58 "github.com/itchyny/base58-go"
	"github.com/pkg/errors"
)

const BadgerSequenceKey = "_links"
const BadgerSequenceLease = 10

type linkShorterBadger struct {
	db  *badger.DB
	seq *badger.Sequence
}

var _ LinkShortner = (*linkShorterBadger)(nil)

func (l *linkShorterBadger) Shrink(url string) (string, error) {
	i, err := l.seq.Next()
	if err != nil {
		return "", errors.Wrap(err, "unable to get next sequence")
	}

	key, err := base58.RippleEncoding.Encode([]byte(strconv.FormatUint(i, 10)))
	if err != nil {
		return "", errors.Wrap(err, "unable to encode to base58")
	}

	err = l.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, []byte(url))
	})

	if err != nil {
		return "", errors.Wrap(err, "unable to add key to database")
	}

	return string(key), nil
}
func (l *linkShorterBadger) Expand(key string) (string, error) {
	var url []byte

	err := l.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte(key))
		if err != nil {
			return err
		}
		url, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get key from database")
	}
	return string(url), nil
}

func (l *linkShorterBadger) Close() error {
	l.seq.Release()
	return l.db.Close()
}

func NewLinkShortnerBadger(dbPath string, opts badger.Options) (LinkShortner, error) {
	opts.Dir = dbPath
	opts.ValueDir = dbPath
	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open database")
	}
	seq, err := db.GetSequence([]byte(BadgerSequenceKey), BadgerSequenceLease)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get sequence from database")
	}
	return &linkShorterBadger{db: db, seq: seq}, nil
}
