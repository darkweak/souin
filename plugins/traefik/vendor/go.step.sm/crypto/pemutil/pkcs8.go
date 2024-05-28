package pemutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"  //nolint:gosec // support for legacy keys
	"crypto/sha1" //nolint:gosec // support for legacy keys
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"hash"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

// PBKDF2SaltSize is the default size of the salt for PBKDF2, 128-bit salt.
const PBKDF2SaltSize = 16

// PBKDF2Iterations is the default number of iterations for PBKDF2.
//
// 600k is the current OWASP recommendation (Dec 2022)
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#pbkdf2
//
// Nist recommends at least 10k (800-63B), 1Password increased in 2023 the
// number of iterations from 100k to 650k.
const PBKDF2Iterations = 600000

// pkcs8 reflects an ASN.1, PKCS#8 PrivateKey. See
// ftp://ftp.rsasecurity.com/pub/pkcs/pkcs-8/pkcs-8v1_2.asn
// and RFC 5208.
type pkcs8 struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
	// optional attributes omitted.
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algo      pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

// Encrypted pkcs8
// Based on https://github.com/youmark/pkcs8
// MIT license
type prfParam struct {
	Algo      asn1.ObjectIdentifier
	NullParam asn1.RawValue
}

type pbkdf2Params struct {
	Salt           []byte
	IterationCount int
	PrfParam       prfParam `asn1:"optional"`
}

type pbkdf2Algorithms struct {
	Algo         asn1.ObjectIdentifier
	PBKDF2Params pbkdf2Params
}

type pbkdf2Encs struct {
	EncryAlgo asn1.ObjectIdentifier
	IV        []byte
}

type pbes2Params struct {
	KeyDerivationFunc pbkdf2Algorithms
	EncryptionScheme  pbkdf2Encs
}

type encryptedlAlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters pbes2Params
}

type encryptedPrivateKeyInfo struct {
	Algo       encryptedlAlgorithmIdentifier
	PrivateKey []byte
}

var (
	// key derivation functions
	oidPKCS5PBKDF2    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidPBES2          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidHMACWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}

	// encryption
	oidAES128CBC = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES192CBC = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 22}
	oidAES256CBC = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	oidDESCBC    = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 7}
	oidD3DESCBC  = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
)

// rfc1423Algo holds a method for enciphering a PEM block.
type rfc1423Algo struct {
	cipher     x509.PEMCipher
	name       string
	cipherFunc func(key []byte) (cipher.Block, error)
	keySize    int
	blockSize  int
	identifier asn1.ObjectIdentifier
}

// rfc1423Algos holds a slice of the possible ways to encrypt a PEM
// block. The ivSize numbers were taken from the OpenSSL source.
var rfc1423Algos = []rfc1423Algo{{
	cipher:     x509.PEMCipherDES,
	name:       "DES-CBC",
	cipherFunc: des.NewCipher,
	keySize:    8,
	blockSize:  des.BlockSize,
	identifier: oidDESCBC,
}, {
	cipher:     x509.PEMCipher3DES,
	name:       "DES-EDE3-CBC",
	cipherFunc: des.NewTripleDESCipher,
	keySize:    24,
	blockSize:  des.BlockSize,
	identifier: oidD3DESCBC,
}, {
	cipher:     x509.PEMCipherAES128,
	name:       "AES-128-CBC",
	cipherFunc: aes.NewCipher,
	keySize:    16,
	blockSize:  aes.BlockSize,
	identifier: oidAES128CBC,
}, {
	cipher:     x509.PEMCipherAES192,
	name:       "AES-192-CBC",
	cipherFunc: aes.NewCipher,
	keySize:    24,
	blockSize:  aes.BlockSize,
	identifier: oidAES192CBC,
}, {
	cipher:     x509.PEMCipherAES256,
	name:       "AES-256-CBC",
	cipherFunc: aes.NewCipher,
	keySize:    32,
	blockSize:  aes.BlockSize,
	identifier: oidAES256CBC,
},
}

func cipherByKey(key x509.PEMCipher) *rfc1423Algo {
	for i := range rfc1423Algos {
		alg := &rfc1423Algos[i]
		if alg.cipher == key {
			return alg
		}
	}
	return nil
}

// deriveKey uses a key derivation function to stretch the password into a key
// with the number of bits our cipher requires. This algorithm was derived from
// the OpenSSL source.
func (c rfc1423Algo) deriveKey(password, salt []byte, h func() hash.Hash) []byte {
	return pbkdf2.Key(password, salt, PBKDF2Iterations, c.keySize, h)
}

// DecryptPEMBlock takes a password encrypted PEM block and the password used
// to encrypt it and returns a slice of decrypted DER encoded bytes.
//
// If the PEM blocks has the Proc-Type header set to "4,ENCRYPTED" it uses
// x509.DecryptPEMBlock to decrypt the block. If not it tries to decrypt the
// block using AES-128-CBC, AES-192-CBC, AES-256-CBC, DES, or 3DES using the
// key derived using PBKDF2 over the given password.
func DecryptPEMBlock(block *pem.Block, password []byte) ([]byte, error) {
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		//nolint:staticcheck // required for legacy compatibility
		return x509.DecryptPEMBlock(block, password)
	}

	// PKCS#8 header defined in RFC7468 section 11
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return DecryptPKCS8PrivateKey(block.Bytes, password)
	}

	return nil, errors.New("unsupported encrypted PEM")
}

// DecryptPKCS8PrivateKey takes a password encrypted private key using the
// PKCS#8 encoding and returns the decrypted data in PKCS#8 form. If an
// incorrect password is detected an x509.IncorrectPasswordError is returned.
// Because of deficiencies in the format, it's not always possible to detect an
// incorrect password. In these cases no error will be returned but the
// decrypted DER bytes will be random noise.
//
// It supports AES-128-CBC, AES-192-CBC, AES-256-CBC, DES, or 3DES encrypted
// data using the key derived with PBKDF2 over the given password.
func DecryptPKCS8PrivateKey(data, password []byte) ([]byte, error) {
	var pki encryptedPrivateKeyInfo
	if _, err := asn1.Unmarshal(data, &pki); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal private key")
	}

	if !pki.Algo.Algorithm.Equal(oidPBES2) {
		return nil, errors.New("unsupported encrypted PEM: only PBES2 is supported")
	}

	if !pki.Algo.Parameters.KeyDerivationFunc.Algo.Equal(oidPKCS5PBKDF2) {
		return nil, errors.New("unsupported encrypted PEM: only PBKDF2 is supported")
	}

	encParam := pki.Algo.Parameters.EncryptionScheme
	kdfParam := pki.Algo.Parameters.KeyDerivationFunc.PBKDF2Params

	iv := encParam.IV
	salt := kdfParam.Salt
	iter := kdfParam.IterationCount

	// pbkdf2 hash function
	keyHash := sha1.New
	if kdfParam.PrfParam.Algo.Equal(oidHMACWithSHA256) {
		keyHash = sha256.New
	}

	var symkey []byte
	var block cipher.Block
	var err error
	switch {
	// AES-128-CBC, AES-192-CBC, AES-256-CBC
	case encParam.EncryAlgo.Equal(oidAES128CBC):
		symkey = pbkdf2.Key(password, salt, iter, 16, keyHash)
		block, err = aes.NewCipher(symkey)
	case encParam.EncryAlgo.Equal(oidAES192CBC):
		symkey = pbkdf2.Key(password, salt, iter, 24, keyHash)
		block, err = aes.NewCipher(symkey)
	case encParam.EncryAlgo.Equal(oidAES256CBC):
		symkey = pbkdf2.Key(password, salt, iter, 32, keyHash)
		block, err = aes.NewCipher(symkey)
	// DES, TripleDES
	case encParam.EncryAlgo.Equal(oidDESCBC):
		symkey = pbkdf2.Key(password, salt, iter, 8, keyHash)
		block, err = des.NewCipher(symkey) //nolint:gosec // support for legacy keys
	case encParam.EncryAlgo.Equal(oidD3DESCBC):
		symkey = pbkdf2.Key(password, salt, iter, 24, keyHash)
		block, err = des.NewTripleDESCipher(symkey) //nolint:gosec // support for legacy keys
	default:
		return nil, errors.Errorf("unsupported encrypted PEM: unknown algorithm %v", encParam.EncryAlgo)
	}
	if err != nil {
		return nil, err
	}

	data = pki.PrivateKey
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)

	// Blocks are padded using a scheme where the last n bytes of padding are all
	// equal to n. It can pad from 1 to blocksize bytes inclusive. See RFC 1423.
	// For example:
	//	[x y z 2 2]
	//	[x y 7 7 7 7 7 7 7]
	// If we detect a bad padding, we assume it is an invalid password.
	blockSize := block.BlockSize()
	dlen := len(data)
	if dlen == 0 || dlen%blockSize != 0 {
		return nil, errors.New("error decrypting PEM: invalid padding")
	}

	last := int(data[dlen-1])
	if dlen < last {
		return nil, x509.IncorrectPasswordError
	}
	if last == 0 || last > blockSize {
		return nil, x509.IncorrectPasswordError
	}
	for _, val := range data[dlen-last:] {
		if int(val) != last {
			return nil, x509.IncorrectPasswordError
		}
	}

	return data[:dlen-last], nil
}

// EncryptPKCS8PrivateKey returns a PEM block holding the given PKCS#8 encroded
// private key, encrypted with the specified algorithm and a PBKDF2 derived key
// from the given password.
func EncryptPKCS8PrivateKey(rand io.Reader, data, password []byte, alg x509.PEMCipher) (*pem.Block, error) {
	ciph := cipherByKey(alg)
	if ciph == nil {
		return nil, errors.Errorf("failed to encrypt PEM: unknown algorithm %v", alg)
	}

	salt := make([]byte, PBKDF2SaltSize)
	if _, err := io.ReadFull(rand, salt); err != nil {
		return nil, errors.Wrap(err, "failed to generate salt")
	}
	iv := make([]byte, ciph.blockSize)
	if _, err := io.ReadFull(rand, iv); err != nil {
		return nil, errors.Wrap(err, "failed to generate IV")
	}

	key := ciph.deriveKey(password, salt, sha256.New)
	block, err := ciph.cipherFunc(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cipher")
	}
	enc := cipher.NewCBCEncrypter(block, iv)
	pad := ciph.blockSize - len(data)%ciph.blockSize
	encrypted := make([]byte, len(data), len(data)+pad)
	// We could save this copy by encrypting all the whole blocks in
	// the data separately, but it doesn't seem worth the additional
	// code.
	copy(encrypted, data)
	// See RFC 1423, section 1.1
	for i := 0; i < pad; i++ {
		encrypted = append(encrypted, byte(pad))
	}
	enc.CryptBlocks(encrypted, encrypted)

	// Build encrypted asn1 data
	pki := encryptedPrivateKeyInfo{
		Algo: encryptedlAlgorithmIdentifier{
			Algorithm: oidPBES2,
			Parameters: pbes2Params{
				KeyDerivationFunc: pbkdf2Algorithms{
					Algo: oidPKCS5PBKDF2,
					PBKDF2Params: pbkdf2Params{
						Salt:           salt,
						IterationCount: PBKDF2Iterations,
						PrfParam: prfParam{
							Algo:      oidHMACWithSHA256,
							NullParam: asn1.NullRawValue,
						},
					},
				},
				EncryptionScheme: pbkdf2Encs{
					EncryAlgo: ciph.identifier,
					IV:        iv,
				},
			},
		},
		PrivateKey: encrypted,
	}

	b, err := asn1.Marshal(pki)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling encrypted key")
	}
	return &pem.Block{
		Type:  "ENCRYPTED PRIVATE KEY",
		Bytes: b,
	}, nil
}
