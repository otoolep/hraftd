package el_test

import (
	"crypto/tls"
	"fmt"
	"image/gif"
	"strings"

	"github.com/pascaldekloe/goe/el"
)

func ExampleAssign_pathAllocation() {
	type node struct {
		Child *node
		Label string
	}

	x := new(node)
	el.Assign(x, "/Child/Child/Label", "Hello")

	fmt.Printf("%v", x.Child.Child)
	// Output: &{<nil> Hello}
}

func ExampleAssign_typeFlexibility() {
	type numbers struct {
		B byte
		I uint32
		F float64
	}

	x := new(numbers)
	el.Assign(x, "/*", 42)

	fmt.Printf("%+v", x)
	// Output: &{B:42 I:42 F:42}
}

func ExampleInt() {
	certPEM := []byte(`-----BEGIN CERTIFICATE-----
MIICADCCAaegAwIBAgIJAN6R4f75XSwGMAkGByqGSM49BAEwODELMAkGA1UEBhMC
TkwxFTATBgNVBAgTDFp1aWQgSG9sbGFuZDESMBAGA1UEChMJUXVpZXMgTmV0MB4X
DTE1MDcxOTE2Mjc1M1oXDTE2MDcxODE2Mjc1M1owODELMAkGA1UEBhMCTkwxFTAT
BgNVBAgTDFp1aWQgSG9sbGFuZDESMBAGA1UEChMJUXVpZXMgTmV0MFkwEwYHKoZI
zj0CAQYIKoZIzj0DAQcDQgAEMpDspvp882GPuJ2/Hk/Ve3z77/QGHDjy3RcAX5gs
0h/6IGX16aR5LQmdIOfhTbJnrcuwspzDAlLHsRRYnFFUiKOBmjCBlzAdBgNVHQ4E
FgQUZsY7o2QXCbSUztcvob5IN3CnKz0waAYDVR0jBGEwX4AUZsY7o2QXCbSUztcv
ob5IN3CnKz2hPKQ6MDgxCzAJBgNVBAYTAk5MMRUwEwYDVQQIEwxadWlkIEhvbGxh
bmQxEjAQBgNVBAoTCVF1aWVzIE5ldIIJAN6R4f75XSwGMAwGA1UdEwQFMAMBAf8w
CQYHKoZIzj0EAQNIADBFAiBw/N+/GobvuxCbzbd84rca4lghmQjw+MQChNO+or7m
+wIhAMONJleFsloA0+PSanth2gRMl3yMlXtRjzmdjcBhoBE6
-----END CERTIFICATE-----`)

	keyPEM := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIL1IxRlbbIhUiIhSfKkksuCaVf2Bl4BQQ9whrOiVWt8voAoGCCqGSM49
AwEHoUQDQgAEMpDspvp882GPuJ2/Hk/Ve3z77/QGHDjy3RcAX5gs0h/6IGX16aR5
LQmdIOfhTbJnrcuwspzDAlLHsRRYnFFUiA==
-----END EC PRIVATE KEY-----`)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	bs, ok := el.Int("/PrivateKey/PublicKey/Curve/CurveParams/BitSize", cert)
	if !ok {
		panic("lookup fail")
	}

	fmt.Printf("size: %d", bs)
	// Output: size: 256
}

func ExampleUints() {
	data := "\x47\x49\x46\x38\x39\x61\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00\x2c\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3b"
	img, err := gif.Decode(strings.NewReader(data))
	if err != nil {
		panic(err)
	}

	fmt.Printf("RGBA: %v", el.Uints("/Palette[0]/*", img))
	// Output: RGBA: [255 255 255 255]
}
