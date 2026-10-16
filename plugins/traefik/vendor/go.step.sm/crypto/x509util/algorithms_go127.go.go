//go:build go1.27

package x509util

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
)

// ML-DSA public key / signature algorithm OIDs (RFC 9881, NIST CSOR).
var (
	oidSignatureMLDSA44 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 17}
	oidSignatureMLDSA65 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 18}
	oidSignatureMLDSA87 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 3, 19}
)

// mldsaSignatureAlgorithms are the ML-DSA signature algorithms, available only
// on Go 1.27 and later. They are appended to baseSignatureAlgorithmMapping.
var mldsaSignatureAlgorithms = []signatureAlgorithmDetail{
	{MLDSA44, x509.MLDSA44, oidSignatureMLDSA44, crypto.Hash(0) /* no pre-hashing */},
	{MLDSA65, x509.MLDSA65, oidSignatureMLDSA65, crypto.Hash(0) /* no pre-hashing */},
	{MLDSA87, x509.MLDSA87, oidSignatureMLDSA87, crypto.Hash(0) /* no pre-hashing */},
}
