package linkedca

// Deduplicate removes duplicate values from the Policy
func (p *Policy) Deduplicate() {
	if p == nil {
		return
	}
	if x509 := p.GetX509(); x509 != nil {
		if allow := x509.GetAllow(); allow != nil {
			allow.Dns = removeDuplicates(allow.Dns)
			allow.Ips = removeDuplicates(allow.Ips)
			allow.Emails = removeDuplicates(allow.Emails)
			allow.Uris = removeDuplicates(allow.Uris)
			allow.CommonNames = removeDuplicates(allow.CommonNames)
		}
		if deny := p.GetX509().GetDeny(); deny != nil {
			deny.Dns = removeDuplicates(deny.Dns)
			deny.Ips = removeDuplicates(deny.Ips)
			deny.Emails = removeDuplicates(deny.Emails)
			deny.Uris = removeDuplicates(deny.Uris)
			deny.CommonNames = removeDuplicates(deny.CommonNames)
		}
	}
	if ssh := p.GetSsh(); ssh != nil {
		if host := ssh.GetHost(); host != nil {
			if allow := host.GetAllow(); allow != nil {
				allow.Dns = removeDuplicates(allow.Dns)
				allow.Ips = removeDuplicates(allow.Ips)
				allow.Principals = removeDuplicates(allow.Principals)
			}
			if deny := host.GetDeny(); deny != nil {
				deny.Dns = removeDuplicates(deny.Dns)
				deny.Ips = removeDuplicates(deny.Ips)
				deny.Principals = removeDuplicates(deny.Principals)
			}
		}
		if user := ssh.GetUser(); user != nil {
			if allow := user.GetAllow(); allow != nil {
				allow.Emails = removeDuplicates(allow.Emails)
				allow.Principals = removeDuplicates(allow.Principals)
			}
			if deny := user.GetDeny(); deny != nil {
				deny.Emails = removeDuplicates(deny.Emails)
				deny.Principals = removeDuplicates(deny.Principals)
			}
		}
	}
}

// removeDuplicates returns a new slice of strings with
// duplicate values removed. It retains the order of elements
// in the source slice.
func removeDuplicates(tokens []string) (ret []string) {
	// no need to remove dupes; return original
	if len(tokens) <= 1 {
		return tokens
	}

	keys := make(map[string]struct{}, len(tokens))

	ret = make([]string, 0, len(tokens))
	for _, item := range tokens {
		if _, ok := keys[item]; ok {
			continue
		}

		keys[item] = struct{}{}
		ret = append(ret, item)
	}

	return
}
