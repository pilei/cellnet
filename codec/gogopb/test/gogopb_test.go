package test

import (
	"reflect"
	"testing"

	"github.com/davyxu/cellnet/codec"
	_ "github.com/davyxu/cellnet/codec/gogopb"
)

func TestGogopbCodec_Codec(t *testing.T) {

	var a ContentACK
	a.Value = 67994
	a.Msg = "hello"

	data, meta, err := codec.EncodeMessage(a, nil)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	outMsg, _, err := codec.DecodeMessage(meta.ID, data)

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if !reflect.DeepEqual(&a, outMsg) {
		t.FailNow()
	}
}
