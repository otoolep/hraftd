package el

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pascaldekloe/goe/verify"
)

type strType string

// strptr returns a pointer to s.
// Go does not allow pointers to literals.
func strptr(s string) *string {
	return &s
}

type Vals struct {
	B bool
	I int64
	U uint64
	F float64
	C complex128
	S string
}

type Ptrs struct {
	BP *bool
	IP *int64
	UP *uint64
	FP *float64
	CP *complex128
	SP *string
}

type Node struct {
	Name  *string
	Child *Node
	child *Node
	X     interface{}
	A     [2]interface{}
	S     []interface{}
}

var testV = Vals{
	B: true,
	I: -2,
	U: 4,
	F: 8,
	C: 16i,
	S: "32",
}

var testPV = Ptrs{
	BP: &testV.B,
	IP: &testV.I,
	UP: &testV.U,
	FP: &testV.F,
	CP: &testV.C,
	SP: &testV.S,
}

type goldenCase struct {
	expr string
	root interface{}
	want interface{}
}

var goldenPaths = []goldenCase{
	0:  {"/B", testV, testV.B},
	1:  {"/IP", &testPV, testV.I},
	2:  {"/X/X/U", Node{X: Node{X: testV}}, testV.U},
	3:  {"/X/../X/FP", Node{X: &testPV}, testV.F},
	4:  {"/X/./X/C", &Node{X: &Node{X: &testV}}, testV.C},
	5:  {"/", &testPV.SP, testV.S},
	6:  {"/.[0]", "hello", uint64('h')},
	7:  {"/S/.[0]", &Node{S: []interface{}{testV.I}}, testV.I},
	8:  {"/A[1]", Node{A: [2]interface{}{testV.F, testV.S}}, testV.S},
	9:  {"/.[true]", map[bool]string{true: "y"}, "y"},
	10: {`/.["I \x2f O"]`, map[strType]float64{"I / O": 99.8}, 99.8},
	11: {"/.[1]/.[2]", map[int]map[uint]string{1: {2: "1.2"}}, "1.2"},
	12: {"/.[*]/.[*]", map[int]map[uint]string{3: {4: "3.4"}}, "3.4"},
}

func TestPaths(t *testing.T) {
	for i, gold := range goldenPaths {
		testGoldenCase(t, reflect.ValueOf(Bool), gold, i)
		testGoldenCase(t, reflect.ValueOf(Int), gold, i)
		testGoldenCase(t, reflect.ValueOf(Uint), gold, i)
		testGoldenCase(t, reflect.ValueOf(Float), gold, i)
		testGoldenCase(t, reflect.ValueOf(Complex), gold, i)
		testGoldenCase(t, reflect.ValueOf(String), gold, i)
	}
}

var goldenPathFails = []goldenCase{
	0:  {"/Name", (*Node)(nil), nil},
	1:  {"/Child", Node{}, nil},
	2:  {"/Child/Name", Node{}, nil},
	3:  {"Malformed", Node{}, nil},
	4:  {"/Mis", Node{}, nil},
	5:  {"/.[broken]", [2]bool{}, nil},
	6:  {"/.[yes]", map[bool]bool{}, nil},
	7:  {"/X", Node{X: testV}, nil},
	8:  {"/.[3]", testV, nil},
	9:  {"/S[4]", Node{}, nil},
	10: {"/A[5]", Node{}, nil},
	11: {"/.[6.66]", map[float64]bool{}, nil},
}

func TestPathFails(t *testing.T) {
	for i, gold := range goldenPathFails {
		testGoldenCase(t, reflect.ValueOf(Bool), gold, i)
		testGoldenCase(t, reflect.ValueOf(Int), gold, i)
		testGoldenCase(t, reflect.ValueOf(Uint), gold, i)
		testGoldenCase(t, reflect.ValueOf(Float), gold, i)
		testGoldenCase(t, reflect.ValueOf(Complex), gold, i)
		testGoldenCase(t, reflect.ValueOf(String), gold, i)
	}
}

func testGoldenCase(t *testing.T, f reflect.Value, gold goldenCase, goldIndex int) {
	args := []reflect.Value{
		reflect.ValueOf(gold.expr),
		reflect.ValueOf(gold.root),
	}
	result := f.Call(args)

	typ := result[0].Type()
	wantMatch := gold.want != nil && typ == reflect.TypeOf(gold.want)

	if got := result[1].Bool(); got != wantMatch {
		t.Errorf("%d: Got %s OK %t, want %t for %q", goldIndex, typ, got, wantMatch, gold.expr)
		return
	}

	if got := result[0].Interface(); wantMatch && got != gold.want {
		t.Errorf("%d: Got %s %#v, want %#v for %q", goldIndex, typ, got, gold.want, gold.expr)
	}
}

func BenchmarkLookups(b *testing.B) {
	todo := b.N
	for {
		for _, g := range goldenPaths {
			String(g.expr, g.root)
			todo--
			if todo == 0 {
				return
			}
		}
	}
}

func TestWildCards(t *testing.T) {
	data := &Node{
		A: [2]interface{}{99, 100},
		S: []interface{}{"a", "b", 3},
	}
	valueMix := []interface{}{testV.B, testV.I, testV.U, testV.F, testV.C, testV.S, testV}

	tests := []struct {
		got, want interface{}
	}{
		0: {Bools("/*", testV), []bool{testV.B}},
		1: {Ints("/*", testV), []int64{testV.I}},
		2: {Uints("/*", testV), []uint64{testV.U}},
		3: {Floats("/*", testV), []float64{testV.F}},
		4: {Complexes("/*", testV), []complex128{testV.C}},
		5: {Strings("/*", testV), []string{testV.S}},

		6:  {Any("/*", Ptrs{}), []interface{}(nil)},
		7:  {Bools("/*", Ptrs{}), []bool(nil)},
		8:  {Ints("/*", Ptrs{}), []int64(nil)},
		9:  {Uints("/*", Ptrs{}), []uint64(nil)},
		10: {Floats("/*", Ptrs{}), []float64(nil)},
		11: {Complexes("/*", Ptrs{}), []complex128(nil)},
		12: {Strings("/*", Ptrs{}), []string(nil)},

		13: {Ints("/A[*]", data), []int64{99, 100}},
		14: {Strings("/*[*]", data), []string{"a", "b"}},

		15: {Any("/.[*]", valueMix), valueMix},
		16: {Any("/", valueMix), []interface{}{valueMix}},
		17: {Any("/MisMatch", valueMix), []interface{}(nil)},
	}

	for i, test := range tests {
		name := fmt.Sprintf("%d: wildcard match", i)
		verify.Values(t, name, test.got, test.want)
	}

}

type goldenAssign struct {
	path  string
	root  interface{}
	value interface{}

	// updates is the wanted number of updates.
	updates int
	// result is the wanted content at path.
	result []string
}

func newGoldenAssigns() []goldenAssign {
	return []goldenAssign{
		{"/", strptr("hello"), "hell", 1, []string{"hell"}},
		{"/.", strptr("hello"), "hell", 1, []string{"hell"}},
		{"/", strptr("hello"), strptr("poin"), 1, []string{"poin"}},

		{"/S", &struct{ S string }{}, "hell", 1, []string{"hell"}},
		{"/SC", &struct{ SC string }{}, strType("hell"), 1, []string{"hell"}},
		{"/CC", &struct{ CC strType }{}, strType("hell"), 1, []string{"hell"}},
		{"/CS", &struct{ CS strType }{}, "hell", 1, []string{"hell"}},

		{"/P", &struct{ P *string }{P: new(string)}, "poin", 1, []string{"poin"}},
		{"/PP", &struct{ PP **string }{PP: new(*string)}, "doub", 1, []string{"doub"}},
		{"/PPP", &struct{ PPP ***string }{PPP: new(**string)}, "trip", 1, []string{"trip"}},

		{"/I", &struct{ I interface{} }{}, "in", 1, []string{"in"}},
		{"/U", &struct{ U interface{} }{U: true}, "up", 1, []string{"up"}},

		{"/X/S", &struct{ X *struct{ S string } }{}, "hell", 1, []string{"hell"}},
		{"/X/P", &struct{ X **struct{ P *string } }{}, "poin", 1, []string{"poin"}},
		{"/X/PP", &struct{ X **struct{ PP **string } }{}, "doub", 1, []string{"doub"}},

		{"/Child/Child/Child/Name", &Node{}, "Grand Grand", 1, []string{"Grand Grand"}},

		{"/.[1]", &[3]*string{}, "up", 1, []string{"up"}},
		{"/.[2]", &[]string{"1", "2", "3"}, "up", 1, []string{"up"}},
		{"/.[3]", &[]*string{}, "in", 1, []string{"in"}},
		{"/.['p']", &map[byte]*string{}, "in", 1, []string{"in"}},
		{"/.['q']", &map[int16]*string{'q': strptr("orig")}, "up", 1, []string{"up"}},
		{"/.['r']", &map[uint]string{}, "in", 1, []string{"in"}},
		{"/.['s']", &map[int64]string{'s': "orig"}, "up", 1, []string{"up"}},
		{"/.[*]", &map[byte]*string{'x': strptr("orig"), 'y': nil}, "up", 2, []string{"up", "up"}},

		{"/.[11]/.[12]", &map[int32]map[int64]string{}, "11.12", 1, []string{"11.12"}},
		{"/.[13]/.[14]", &map[int8]**map[int16]string{}, "13.14", 1, []string{"13.14"}},
		{"/.['w']/X/Y", &map[byte]struct{ X struct{ Y ***string } }{}, "z", 1, []string{"z"}},
	}
}

func newGoldenAssignFails() []goldenAssign {
	return []goldenAssign{
		// No expression
		{"", strptr("hello"), "fail", 0, nil},

		// Nil root
		{"/", nil, "fail", 0, nil},

		// Nil value
		{"/", strptr("hello"), nil, 0, []string{"hello"}},

		// Not addresable
		{"/", "hello", "fail", 0, []string{"hello"}},

		// Too abstract
		{"/X/anyField", &Node{}, "fail", 0, nil},

		// Wrong type
		{"/Sp", &struct{ Sp *string }{}, 9.98, 0, []string{""}},

		// String modification
		{"/.[6]", strptr("immutable"), '-', 0, nil},

		// Out of bounds
		{"/.[8]", &[2]string{}, "fail", 0, nil},

		// Malformed map keys
		{"/Sk[''']", &struct{ Sk map[string]string }{}, "fail", 0, nil},
		{"/Ik[''']", &struct{ Ik map[int]string }{}, "fail", 0, nil},
		{"/Ik[z]", &struct{ Ik map[int]string }{}, "fail", 0, nil},
		{"/Uk[''']", &struct{ Uk map[uint]string }{}, "fail", 0, nil},
		{"/Uk[z]", &struct{ Uk map[uint]string }{}, "fail", 0, nil},
		{"/Fk[z]", &struct{ Fk map[float32]string }{}, "fail", 0, nil},
		{"/Ck[z]", &struct{ Ck map[complex128]string }{}, "fail", 0, nil},
		{"/Ck[]", &struct{ Ck map[complex128]string }{}, "fail", 0, nil},

		// Non-exported
		{`/child/Name`, &Node{}, "fail", 0, nil},
		{`/ns`, &struct{ ns *string }{}, "fail", 0, nil},

		// Non-exported array
		{`/na[0]`, &struct{ na [2]string }{}, "fail", 0, []string{""}},
		{`/na[1]`, &struct{ na [2]*string }{}, "fail", 0, nil},
		{`/na[*]`, &struct{ na [2]string }{}, "fail", 0, []string{"", ""}},

		// Non-exported slice
		{`/ns[0]`, &struct{ ns []string }{}, "fail", 0, nil},
		{`/ns[1]`, &struct{ ns []*string }{ns: []*string{nil, strptr("b")}}, "fail", 0, []string{"b"}},
		{`/ns[*]`, &struct{ ns []string }{ns: []string{"a"}}, "fail", 0, []string{"a"}},

		// Non-exported map
		{`/nm[0]`, &struct{ nm map[int]string }{}, "fail", 0, nil},
		{`/nm[1]`, &struct{ nm map[int]*string }{nm: map[int]*string{1: strptr("b")}}, "fail", 0, []string{"b"}},
		{`/nm[*]`, &struct{ nm map[int]string }{nm: map[int]string{2: "c"}}, "fail", 0, []string{"c"}},
	}
}

func TestAssigns(t *testing.T) {
	for _, gold := range append(newGoldenAssigns(), newGoldenAssignFails()...) {
		n := Assign(gold.root, gold.path, gold.value)
		if n != gold.updates {
			t.Errorf("Got n=%d, want %d for %s", n, gold.updates, gold.path)
		}

		got := Strings(gold.path, gold.root)
		verify.Values(t, gold.path, got, gold.result)
	}
}

func BenchmarkAssigns(b *testing.B) {
	b.StopTimer()
	todo := b.N
	for {
		cases := newGoldenAssigns()
		b.StartTimer()
		for _, g := range cases {
			Assign(g.root, g.path, g.value)
			todo--
			if todo == 0 {
				return
			}
		}
		b.StopTimer()
	}
}
