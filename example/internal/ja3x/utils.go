package ja3x

import (
	"github.com/dreadl0ck/ja3"
	"github.com/dreadl0ck/tlsx"
)

// clientHelloDigest generates a ja3 digest from the bytes
// of a given TLS Client Hello message.
func clientHelloDigest(raw []byte) (string, error) {
	chb := &tlsx.ClientHelloBasic{}
	if err := chb.Unmarshal(raw); err != nil {
		return "", err
	}
	return ja3.BareToDigestHex(ja3.Bare(chb)), nil
}
