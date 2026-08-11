package asn1utils

// IsPrintableString reports whether the given s is a valid ASN.1 PrintableString.
// If asterisk is allowAsterisk then '*' is also allowed, reflecting existing
// practice. If ampersand is allowAmpersand then '&' is allowed as well.
func IsPrintableString(s string, asterisk, ampersand bool) bool {
	for _, b := range s {
		valid := 'a' <= b && b <= 'z' ||
			'A' <= b && b <= 'Z' ||
			'0' <= b && b <= '9' ||
			'\'' <= b && b <= ')' ||
			'+' <= b && b <= '/' ||
			b == ' ' ||
			b == ':' ||
			b == '=' ||
			b == '?' ||
			// This is technically not allowed in a PrintableString.
			// However, x509 certificates with wildcard strings don't
			// always use the correct string type so we permit it.
			(asterisk && b == '*') ||
			// This is not technically allowed either. However, not
			// only is it relatively common, but there are also a
			// handful of CA certificates that contain it. At least
			// one of which will not expire until 2027.
			(ampersand && b == '&')

		if !valid {
			return false
		}
	}

	return true
}
