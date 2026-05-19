package jose

import (
	fileutils "go.step.sm/crypto/internal/utils/file"
)

type context struct {
	filename         string
	use, alg, kid    string
	subtle, insecure bool
	noDefaults       bool
	password         []byte
	passwordPrompt   string
	passwordPrompter PasswordPrompter
	contentType      string
}

// apply the options to the context and returns an error if one of the options
// fails.
func (ctx *context) apply(opts ...Option) (*context, error) {
	for _, opt := range opts {
		if err := opt(ctx); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// Option is the type used to add attributes to the context.
type Option func(ctx *context) error

// WithFilename adds the given filename to the context.
func WithFilename(filename string) Option {
	return func(ctx *context) error {
		ctx.filename = filename
		return nil
	}
}

// WithUse adds the use claim to the context.
func WithUse(use string) Option {
	return func(ctx *context) error {
		ctx.use = use
		return nil
	}
}

// WithAlg adds the alg claim to the context.
func WithAlg(alg string) Option {
	return func(ctx *context) error {
		ctx.alg = alg
		return nil
	}
}

// WithKid adds the kid property to the context.
func WithKid(kid string) Option {
	return func(ctx *context) error {
		ctx.kid = kid
		return nil
	}
}

// WithSubtle marks the context as subtle.
func WithSubtle(subtle bool) Option {
	return func(ctx *context) error {
		ctx.subtle = subtle
		return nil
	}
}

// WithInsecure marks the context as insecure.
func WithInsecure(insecure bool) Option {
	return func(ctx *context) error {
		ctx.insecure = insecure
		return nil
	}
}

// WithNoDefaults avoids that the parser loads defaults values, specially the
// default algorithms.
func WithNoDefaults(val bool) Option {
	return func(ctx *context) error {
		ctx.noDefaults = val
		return nil
	}
}

// WithPassword is a method that adds the given password to the context.
func WithPassword(pass []byte) Option {
	return func(ctx *context) error {
		ctx.password = pass
		return nil
	}
}

// WithPasswordFile is a method that adds the password in a file to the context.
func WithPasswordFile(filename string) Option {
	return func(ctx *context) error {
		b, err := fileutils.ReadPasswordFromFile(filename)
		if err != nil {
			return err
		}
		ctx.password = b
		return nil
	}
}

// WithPasswordPrompter defines a method that can be used to prompt for the
// password to decrypt an encrypted JWE.
func WithPasswordPrompter(prompt string, fn PasswordPrompter) Option {
	return func(ctx *context) error {
		ctx.passwordPrompt = prompt
		ctx.passwordPrompter = fn
		return nil
	}
}

// WithContentType adds the content type when encrypting data.
func WithContentType(cty string) Option {
	return func(ctx *context) error {
		ctx.contentType = cty
		return nil
	}
}
