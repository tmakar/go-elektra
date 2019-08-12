package kdb

// #include <elektra/kdb.h>
// #include <stdlib.h>
//
// static Key * keyNewEmptyWrapper() {
//   return keyNew(0);
// }
//
// static Key * keyNewWrapper(char* k) {
//   return keyNew(k, KEY_END);
// }
//
// static Key * keyNewValueWrapper(char* k, char* v) {
//   return keyNew(k, KEY_VALUE, v, KEY_END);
// }
import "C"

import (
	"fmt"
	"unsafe"
	"runtime"

	"github.com/pkg/errors"
)

type Key interface {
	// Duplicate() Key

	BaseName() string
	Name() string
	Value() string
	Boolean() bool
	Bytes() []byte
	Meta(name string) string

	SetMeta(name, value string) error
	SetName(name string) error
	SetBoolean(value bool) error
	SetString(value string) error
	SetBytes(value []byte) error
}

type ckey struct {
	key *C.struct__Key
}

// CreateKey creates a new key with an optional value.
func CreateKey(name string, value ...interface{}) (Key, error) {
	var key *ckey

	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))

	if name == "" {
		key = newKey(C.keyNewEmptyWrapper())
	} else if len(value) > 0 {
		switch v := value[0].(type) {
		case string:
			cValue := C.CString(v)
			key = newKey(C.keyNewValueWrapper(n, cValue))
			defer C.free(unsafe.Pointer(cValue))
		default:
			return nil, errors.New("unsupported key value type")
		}
	} else {
		key = newKey(C.keyNewWrapper(n))
	}

	if key.key == nil {
		return nil, errors.New("could not create key")
	}

	runtime.SetFinalizer(key, freeKey)

	return key, nil
}

func freeKey(k *ckey) {
	k.free()
}

func newKey(k *C.struct__Key) *ckey {
	return &ckey{k}
}

func toCKey(key Key) (*ckey, error) {
	if key == nil {
		return nil, errors.New("key is nil")
	}

	ckey, ok := key.(*ckey)

	if !ok {
		return nil, errors.New("only pointer to ckey struct allowed")
	}

	return ckey, nil
}

// BaseName returns the basename of the Key.
func (k *ckey) BaseName() string {
	name := C.keyBaseName(k.key)

	return C.GoString(name)
}

// Name returns the name of the Key.
func (k *ckey) Name() string {
	name := C.keyName(k.key)

	return C.GoString(name)
}

// free frees the resources of the Key.
func (k *ckey) free() {
	if !k.isNil() {
		C.keyDel(k.key)
	}
}

// Boolean returns the boolean value of the Key.
func (k *ckey) Boolean() bool {
	return k.Value() == "1"
}

// SetBytes sets the value of a key to a byte slice.
func (k *ckey) SetBytes(value []byte) error {
	v := C.CString(string(value))
	defer C.free(unsafe.Pointer(v))

	size := C.ulong(len(value))

	_ = C.keySetBinary(k.key, unsafe.Pointer(v), size)

	return nil
}

// SetString sets the value of a key to a string.
func (k *ckey) SetString(value string) error {
	v := C.CString(value)
	defer C.free(unsafe.Pointer(v))

	_ = C.keySetString(k.key, v)

	return nil
}

// SetBoolean sets the value of a key to a boolean
// where true is represented as "1" and false as "0".
func (k *ckey) SetBoolean(value bool) error {
	strValue := "0"

	if value {
		strValue = "1"
	}

	return k.SetString(strValue)
}

// SetName sets the name of the Key.
func (k *ckey) SetName(name string) error {
	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))

	if ret := C.keySetName(k.key, n); ret < 0 {
		return errors.New("could not set key name")
	}

	return nil
}

// Bytes returns the value of the Key as a byte slice.
func (k *ckey) Bytes() []byte {
	ptr := (*C.char)(C.keyValue(k.key))

	v := C.GoString(ptr)

	return []byte(v)
}

// Value returns the string value of the Key.
func (k *ckey) Value() string {
	str := C.keyString(k.key)

	return C.GoString(str)
}


// String returns the string representation of the Key
// in "Key: Value" format.
func (k *ckey) String() string {
	name := k.Name()
	value := k.Value()

	if value == "" {
		value = "(empty)"
	}

	return fmt.Sprintf("%s: %s", name, value) 
}

// SetMeta sets the meta value of a Key.
func (k *ckey) SetMeta(name, value string) error {
	cName, cValue := C.CString(name), C.CString(value)

	defer C.free(unsafe.Pointer(cName))
	defer C.free(unsafe.Pointer(cValue))

	ret := C.keySetMeta(k.key, cName, cValue)

	if ret < 0 {
		return errors.New("could not set meta")
	}

	return nil
}

// Meta retrieves the Meta value of a Key.
func (k *ckey) Meta(name string) string {
	cName := C.CString(name)

	defer C.free(unsafe.Pointer(cName))

	metaKey := newKey(C.keyGetMeta(k.key, cName))

	if metaKey.isNil() {
		return ""
	}

	return metaKey.Value()
}

func (k *ckey) isNil() bool {
	return k.key == nil
}
