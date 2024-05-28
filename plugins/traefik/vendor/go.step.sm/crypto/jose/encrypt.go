package jose

import (
	"encoding/json"

	"github.com/pkg/errors"
	"go.step.sm/crypto/randutil"
)

// MaxDecryptTries is the maximum number of attempts to decrypt a file.
const MaxDecryptTries = 3

// PasswordPrompter defines the function signature for the PromptPassword
// callback.
type PasswordPrompter func(s string) ([]byte, error)

// PromptPassword is a method used to prompt for a password to decode encrypted
// keys. If this method is not defined and the key or password are not passed,
// the parse of the key will fail.
var PromptPassword PasswordPrompter

// Encrypt returns the given data encrypted with the default encryption
// algorithm (PBES2-HS256+A128KW).
func Encrypt(data []byte, opts ...Option) (*JSONWebEncryption, error) {
	ctx, err := new(context).apply(opts...)
	if err != nil {
		return nil, err
	}

	var passphrase []byte
	switch {
	case len(ctx.password) > 0:
		passphrase = ctx.password
	case ctx.passwordPrompter != nil:
		if passphrase, err = ctx.passwordPrompter(ctx.passwordPrompt); err != nil {
			return nil, err
		}
	case PromptPassword != nil:
		if passphrase, err = PromptPassword("Please enter the password to encrypt the data"); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("failed to encrypt the data: missing password")
	}

	salt, err := randutil.Salt(PBKDF2SaltSize)
	if err != nil {
		return nil, err
	}

	// Encrypt private key using PBES2
	recipient := Recipient{
		Algorithm:  PBES2_HS256_A128KW,
		Key:        passphrase,
		PBES2Count: PBKDF2Iterations,
		PBES2Salt:  salt,
	}

	encrypterOptions := new(EncrypterOptions)
	if ctx.contentType != "" {
		encrypterOptions.WithContentType(ContentType(ctx.contentType))
	}

	encrypter, err := NewEncrypter(DefaultEncAlgorithm, recipient, encrypterOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error creating cipher")
	}

	jwe, err := encrypter.Encrypt(data)
	if err != nil {
		return nil, errors.Wrap(err, "error encrypting data")
	}

	return jwe, nil
}

// EncryptJWK returns the given JWK encrypted with the default encryption
// algorithm (PBES2-HS256+A128KW).
func EncryptJWK(jwk *JSONWebKey, passphrase []byte) (*JSONWebEncryption, error) {
	b, err := json.Marshal(jwk)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling JWK")
	}

	return Encrypt(b, WithPassword(passphrase), WithContentType("jwk+json"))
}

// Decrypt returns the decrypted version of the given data if it's encrypted,
// it will return the raw data if it's not encrypted or the format is not
// valid.
func Decrypt(data []byte, opts ...Option) ([]byte, error) {
	ctx, err := new(context).apply(opts...)
	if err != nil {
		return nil, err
	}

	enc, err := ParseEncrypted(string(data))
	if err != nil {
		return data, nil //nolint:nilerr // Return the given data if we cannot parse it as encrypted.
	}

	// Try with the given password.
	if len(ctx.password) > 0 {
		if data, err = enc.Decrypt(ctx.password); err == nil {
			return data, nil
		}
		return nil, errors.New("failed to decrypt JWE: invalid password")
	}

	// Try with a given password prompter.
	if ctx.passwordPrompter != nil || PromptPassword != nil {
		var pass []byte
		for i := 0; i < MaxDecryptTries; i++ {
			switch {
			case ctx.passwordPrompter != nil:
				if pass, err = ctx.passwordPrompter(ctx.passwordPrompt); err != nil {
					return nil, err
				}
			case ctx.filename != "":
				if pass, err = PromptPassword("Please enter the password to decrypt " + ctx.filename); err != nil {
					return nil, err
				}
			default:
				if pass, err = PromptPassword("Please enter the password to decrypt the JWE"); err != nil {
					return nil, err
				}
			}
			if data, err = enc.Decrypt(pass); err == nil {
				return data, nil
			}
		}
	}

	return nil, errors.New("failed to decrypt JWE: invalid password")
}
