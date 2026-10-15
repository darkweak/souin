//go:build !go1.27

package x509util

// mldsaSignatureAlgorithms is empty on Go toolchains older than 1.27, where the
// standard library does not define the ML-DSA signature algorithms. Attempting
// to use an ML-DSA algorithm name will fail with an "unsupported
// signatureAlgorithm" error.
var mldsaSignatureAlgorithms []signatureAlgorithmDetail
