package action

import (
	"fmt"

	"github.com/indexexchange/haproxy-spoe-go/typeddata"
	"github.com/indexexchange/haproxy-spoe-go/varint"
)

// Marshal appends the wire encoding of the action to buf and returns the
// extended buffer. buf is returned (possibly grown) even on error, so
// callers reusing a pooled buffer never lose it.
func (action *Action) Marshal(buf []byte) ([]byte, error) {
	var nb byte

	switch action.Type {
	case TypeSetVar:
		nb = nbVarsSetVar
	case TypeUnsetVar:
		nb = nbVarsUnsetVar
	default:
		return buf, fmt.Errorf("unexpected action type: %v", action.Type)
	}

	buf = append(buf, byte(action.Type), nb, byte(action.Scope))

	var b [10]byte
	n := varint.PutUvarint(b[:], uint64(len(action.Name)))

	buf = append(buf, b[:n]...)
	buf = append(buf, action.Name...)

	var err error
	buf, _, err = typeddata.Encode(action.Value, buf)

	return buf, err
}
