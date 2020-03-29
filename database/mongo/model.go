package mongo

import (
	"log"
	"time"

	"github.com/globalsign/mgo"
)

type (
	options struct {
		timeout time.Duration
	}

	Option func(opts *options)

	Model struct {
		session    *concurrentSession
		db         *mgo.Database
		collection string
		opts       []Option
	}
)

func MustNewModel(url, database, collection string, opts ...Option) *Model {
	model, err := NewModel(url, database, collection, opts...)
	if err != nil {
		log.Fatal(err)
	}

	return model
}

func NewModel(url, database, collection string, opts ...Option) (*Model, error) {
	session, err := getConcurrentSession(url)
	if err != nil {
		return nil, err
	}

	return &Model{
		session:    session,
		db:         session.DB(database),
		collection: collection,
		opts:       opts,
	}, nil
}

func (mm *Model) GetCollection(session *mgo.Session) Collection {
	return newCollection(mm.db.C(mm.collection).With(session))
}

func (mm *Model) PutSession(session *mgo.Session) {
	mm.session.putSession(session)
}

func (mm *Model) TakeSession() (*mgo.Session, error) {
	return mm.session.takeSession(mm.opts...)
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *options) {
		opts.timeout = timeout
	}
}
