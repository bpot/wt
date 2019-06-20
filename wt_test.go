package wt

import (
	"bytes"
	"fmt"
	"testing"
)

func TestConstruction(t *testing.T) {
	text := []byte("bananaz")
	wt, err := New(text)
	if err != nil {
		t.Fatal(err)
	}

	b := wt.Access(0)
	if 'b' != b {
		t.Errorf("expected %q; got %q", 'b', b)
	}

	b = wt.Access(1)
	if 'a' != b {
		t.Errorf("expected %q; got %q", 'a', b)
	}

	b = wt.Access(2)
	if 'n' != b {
		t.Errorf("expected %q; got %q", 'n', b)
	}

	extract := []byte{}
	for i := 0; i < 7; i++ {
		extract = append(extract, wt.Access(uint64(i)))
	}
	if !bytes.Equal(text, extract) {
		t.Errorf("expected %q; got %q", text, extract)
	}
}

func TestAccess(t *testing.T) {
	bwt := []byte("ipssm\x00pissii")
	wt, err := New(bwt)
	if err != nil {
		t.Fatal(err)
	}

	b := wt.Access(0)
	if 'i' != b {
		t.Errorf("expected %q; got %q", 'i', b)
	}
	recreated := []byte{}
	for i := 0; i < len(bwt); i++ {
		fmt.Println(i)
		recreated = append(recreated, wt.Access(uint64(i)))
	}

	if !bytes.Equal(bwt, recreated) {
		t.Errorf("expected %q;got %q", bwt, recreated)
	}

	r := wt.Rank('i', 10)
	if 2 != r {
		t.Errorf("expected %d; got %d", 2, r)
	}

	wt.Rank('m', 4)
}

func TestRank(t *testing.T) {
	wt, err := New([]byte("bananaz"))
	if err != nil {
		t.Fatal(err)
	}

	r := wt.Rank('a', 0)
	if 0 != r {
		t.Errorf("expected 0; got %d", r)
	}
	r = wt.Rank('a', 1)
	if 0 != r {
		t.Errorf("expected 0; got %d", r)
	}
	r = wt.Rank('a', 2)
	if 1 != r {
		t.Errorf("expected 1; got %d", r)
	}
	r = wt.Rank('b', 2)
	if 1 != r {
		t.Errorf("expected 1; got %d", r)
	}
	r = wt.Rank('n', 5)
	if 2 != r {
		t.Errorf("expected 2; got %d", r)
	}

	r = wt.Rank('o', 5)
	if 0 != r {
		t.Errorf("expected 0; got %d", r)
	}
}

func TestAccessMore(t *testing.T) {
	bwt := []byte("\r080 017-:\x00 1:00481")
	wt, err := New(bwt)
	if err != nil {
		t.Fatal(err)
	}
	b := wt.Access(16)
	if bwt[16] != b {
		t.Errorf("expected %q; got %q", bwt[16], b)
	}
	recreated := []byte{}
	for i := 0; i < len(bwt); i++ {
		fmt.Println(i)
		recreated = append(recreated, wt.Access(uint64(i)))
	}
	if !bytes.Equal(bwt, recreated) {
		t.Errorf("expected %q;got %q", bwt, recreated)
	}
}

func TestAccessMoreMore(t *testing.T) {
	bwt := []byte("F\r\r>080r:b xt017-:\x00 1:004o81 mk \n \nfoen@a.cth<trooiTBocbp.Pbbeeotooo")
	wt, err := New(bwt)
	if err != nil {
		t.Fatal(err)
	}
	b := wt.Access(1)
	if bwt[1] != b {
		t.Errorf("expected %q; got %q", bwt[1], b)
	}

	var buf bytes.Buffer
	err = wt.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	wtRT, wtBytes, err := NewFromSerialized(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if buf.Len() != wtBytes {
		t.Errorf("expected %d; got %d", buf.Len(), wtBytes)
	}

	recreated := []byte{}
	for i := 0; i < len(bwt); i++ {
		recreated = append(recreated, wtRT.Access(uint64(i)))
	}
	if !bytes.Equal(bwt, recreated) {
		t.Errorf("expected %q;got %q", bwt, recreated)
	}
}
