package message

import (
	"fmt"
	"testing"
)

var Expected = []string{
	`
	Segments{
		Segment{ DStart: 0, DEnd: 192, PEnd: 256, SLen: 123, Last: 123},
		Segment{ DStart: 256, DEnd: 448, PEnd: 512, SLen: 123, Last: 123},
		Segment{ DStart: 512, DEnd: 704, PEnd: 768, SLen: 123, Last: 123},
		Segment{ DStart: 768, DEnd: 960, PEnd: 1024, SLen: 123, Last: 123},
		Segment{ DStart: 1024, DEnd: 1216, PEnd: 1280, SLen: 123, Last: 123},
		Segment{ DStart: 1280, DEnd: 1472, PEnd: 1536, SLen: 123, Last: 123},
		Segment{ DStart: 1536, DEnd: 1728, PEnd: 1792, SLen: 123, Last: 123},
		Segment{ DStart: 1792, DEnd: 1984, PEnd: 2048, SLen: 123, Last: 123},
		Segment{ DStart: 2048, DEnd: 2240, PEnd: 2304, SLen: 123, Last: 123},
		Segment{ DStart: 2304, DEnd: 2496, PEnd: 2560, SLen: 123, Last: 123},
		Segment{ DStart: 2560, DEnd: 2752, PEnd: 2816, SLen: 123, Last: 123},
		Segment{ DStart: 2816, DEnd: 2837, PEnd: 2844, SLen: 123, Last: 19},
	}
`,
	`
	Segments{
		Segment{ DStart: 0, DEnd: 133, PEnd: 133, SLen: 3963, Last: 1172},
	}
`,
	`
	Segments{
		Segment{ DStart: 0, DEnd: 128, PEnd: 256, SLen: 3963, Last: 3963},
		Segment{ DStart: 256, DEnd: 261, PEnd: 266, SLen: 3963, Last: 1172},
	}
`,
	`
	Segments{
		Segment{ DStart: 0, DEnd: 67, PEnd: 67, SLen: 3963, Last: 586},
	}
`,
}

func TestNewSegments(t *testing.T) {
	msgSize := 2<<17 + 111
	segSize := 256
	s := NewSegments(msgSize, segSize, Overhead, 64)
	o := fmt.Sprint(s)
	if o != Expected[0] {
		t.Errorf(
			"Failed to correctly generate.\ngot:\n'%s'\nexpected:\n'%s'",
			o, Expected[0])
	}
	msgSize = 2 << 18
	segSize = 4096
	s = NewSegments(msgSize, segSize, Overhead, 0)
	o = fmt.Sprint(s)
	if o != Expected[1] {
		t.Errorf(
			"Failed to correctly generate.\ngot:\n%s\nexpected:\n%s",
			o, Expected[0])
	}
	s = NewSegments(msgSize, segSize, Overhead, 128)
	o = fmt.Sprint(s)
	if o != Expected[2] {
		t.Errorf(
			"Failed to correctly generate.\ngot:\n%s\nexpected:\n%s",
			o, Expected[0])
	}
	msgSize = 2 << 17
	segSize = 4096
	s = NewSegments(msgSize, segSize, Overhead, 0)
	o = fmt.Sprint(s)
	if o != Expected[3] {
		t.Errorf(
			"Failed to correctly generate.\ngot:\n%s\nexpected:\n%s",
			o, Expected[0])
	}
}
