//nolint:revive // ignore mocked methods for unsupported DB type
package database

// NotSupportedDB is a db implementation used on database drivers when the
// no<driver> tags are used.
type NotSupportedDB struct{}

func (*NotSupportedDB) Open(dataSourceName string, opt ...Option) error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) Close() error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) Get(bucket, key []byte) (ret []byte, err error) {
	return nil, ErrOpNotSupported
}

func (*NotSupportedDB) Set(bucket, key, value []byte) error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) CmpAndSwap(bucket, key, oldValue, newValue []byte) ([]byte, bool, error) {
	return nil, false, ErrOpNotSupported
}

func (*NotSupportedDB) Del(bucket, key []byte) error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) List(bucket []byte) ([]*Entry, error) {
	return nil, ErrOpNotSupported
}

func (*NotSupportedDB) Update(tx *Tx) error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) CreateTable(bucket []byte) error {
	return ErrOpNotSupported
}

func (*NotSupportedDB) DeleteTable(bucket []byte) error {
	return ErrOpNotSupported
}
