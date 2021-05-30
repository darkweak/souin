package ykeys

import (
	"fmt"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

const FIRST_KEY = "The_First_Test"
const SECOND_KEY = "The_Second_Test"
const THIRD_KEY = "The_Third_Test"
const FOURTH_KEY = "The_Fourth_Test"

func mockYkeys() map[string]configurationtypes.YKey {
	var config configuration.Configuration
	_ = config.Parse([]byte(
		`
ykeys:
  The_First_Test:
    headers:
      Authorization: '.+'
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
  The_Third_Test:
  The_Fourth_Test:
`))
	return config.Ykeys
}

func TestInitializeYKeys(t *testing.T) {
	r := InitializeYKeys(mockYkeys())

	if nil == r || nil == r.Cache {
		errors.GenerateError(t, "Ristretto should be instanciated")
	}

	if nil == r || nil == r.keySaver {
		errors.GenerateError(t, "The key saver should be instanciated")
	}
}

func TestYKeyStorage_AddToTags(t *testing.T) {
	baseYkeys := mockYkeys()
	url := "http://the.url.com"

	r := InitializeYKeys(baseYkeys)
	r.AddToTags(url, []string{FIRST_KEY})
	time.Sleep(200*time.Millisecond)
	res, _ := r.Cache.Get(FIRST_KEY)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R1 => The key %s should contains the url %s, %s given", FIRST_KEY, url, res))
	}

	r.AddToTags(url, []string{FIRST_KEY, SECOND_KEY})
	r.AddToTags("http://domain.com", []string{FIRST_KEY})

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(FIRST_KEY)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains the url %s, %s given", FIRST_KEY, url, res))
	}
	if len(strings.Split(res.(string), ",")) != 2 {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains 2 records, %d given", FIRST_KEY, len(strings.Split(res.(string), ","))))
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(SECOND_KEY)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains the url %s, %s given", SECOND_KEY, url, res))
	}
	if len(strings.Split(res.(string), ",")) != 1 {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains 2 records, %d given", FIRST_KEY, len(strings.Split(res.(string), ","))))
	}

	r.AddToTags(url, []string{FIRST_KEY})
	r.AddToTags(url, []string{FIRST_KEY})
	r.AddToTags("http://domain.com", []string{FIRST_KEY})
	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(FIRST_KEY)
	if len(strings.Split(res.(string), ",")) != 2 {
		errors.GenerateError(t, fmt.Sprintf("R3 => The key %s should contains 2 records, %d given", FIRST_KEY, len(strings.Split(res.(string), ","))))
	}
}

func TestYKeyStorage_InvalidateTags(t *testing.T) {
	baseYkeys := mockYkeys()
	url1 := "http://the.url1.com"
	url2 := "http://the.url2.com"
	url3 := "http://the.url3.com"
	url4 := "http://the.url4.com"

	r := InitializeYKeys(baseYkeys)
	r.AddToTags(url1, []string{FIRST_KEY, SECOND_KEY})
	r.AddToTags(url2, []string{FIRST_KEY, SECOND_KEY})
	r.AddToTags(url3, []string{FIRST_KEY, THIRD_KEY})
	r.AddToTags(url4, []string{THIRD_KEY, FOURTH_KEY})

	urls := r.InvalidateTags([]string{FIRST_KEY})

	if len(urls) != 3 {
		errors.GenerateError(t, fmt.Sprintf("It should have 3 urls to remove (e.g. [http://the.url1.com http://the.url2.com http://the.url3.com]), %v given", urls))
	}

	time.Sleep(200*time.Millisecond)
	res, _ := r.Cache.Get(FIRST_KEY)
	if res.(string) != "" {
		errors.GenerateError(t, "The FIRST_KEY should be empty")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(SECOND_KEY)
	if res.(string) != "" {
		errors.GenerateError(t, "The SECOND_KEY should be empty")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(THIRD_KEY)
	if res.(string) != url4 {
		errors.GenerateError(t, "The THIRD_KEY should be equals to http://the.url4.com")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(FOURTH_KEY)
	if res.(string) != url4 {
		errors.GenerateError(t, "The FOURTH_KEY should be equals to http://the.url4.com")
	}
}

func TestYKeyStorage_InvalidateTagURLs(t *testing.T) {
	baseYkeys := mockYkeys()
	url1 := "http://the.url1.com"
	url2 := "http://the.url2.com"
	url3 := "http://the.url3.com"
	url4 := "http://the.url4.com"

	r := InitializeYKeys(baseYkeys)
	r.AddToTags(url1, []string{FIRST_KEY, SECOND_KEY})
	r.AddToTags(url2, []string{FIRST_KEY, SECOND_KEY})
	r.AddToTags(url3, []string{FIRST_KEY, THIRD_KEY})
	r.AddToTags(url4, []string{THIRD_KEY, FOURTH_KEY})

	urls := r.InvalidateTagURLs(fmt.Sprintf("%s,%s", url1, url3))
	if len(urls) != 2 {
		errors.GenerateError(t, "It should have 2 urls to remove (e.g. [http://the.url1.com http://the.url3.com])")
	}

	time.Sleep(200*time.Millisecond)
	res, _ := r.Cache.Get(FIRST_KEY)
	if res.(string) != url2 {
		errors.GenerateError(t, "The FIRST_KEY should be equals to http://the.url2.com")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(SECOND_KEY)
	if res.(string) != url2 {
		errors.GenerateError(t, "The SECOND_KEY should be equals to http://the.url2.com")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(THIRD_KEY)
	if res.(string) != url4 {
		errors.GenerateError(t, "The THIRD_KEY should be equals to http://the.url4.com")
	}

	time.Sleep(200*time.Millisecond)
	res, _ = r.Cache.Get(FOURTH_KEY)
	if res.(string) != url4 {
		errors.GenerateError(t, "The FOURTH_KEY should be equals to http://the.url4.com")
	}
}

func TestYKeyStorage_GetValidatedTags(t *testing.T) {
	baseYkeys := mockYkeys()
	r := InitializeYKeys(baseYkeys)
	baseUrl := "http://the.url.com"
	invalidUrl := "http://domain.com/test/the/second"
	validUrl := invalidUrl + "/anything"
	if len(r.GetValidatedTags(baseUrl, nil)) != 2 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseUrl, []string{THIRD_KEY, FOURTH_KEY}, r.GetValidatedTags(baseUrl, nil)))
	}
	if len(r.GetValidatedTags(invalidUrl, nil)) != 2 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseUrl, []string{THIRD_KEY, FOURTH_KEY}, r.GetValidatedTags(baseUrl, nil)))
	}
	if len(r.GetValidatedTags(validUrl, nil)) != 3 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseUrl, []string{SECOND_KEY, THIRD_KEY, FOURTH_KEY}, r.GetValidatedTags(baseUrl, nil)))
	}

	var headers http.Header
	headers = http.Header{}
	headers.Set("Authorization", "anything")
	headers.Set("Content-Type", "any value here")
	if len(r.GetValidatedTags(baseUrl, headers)) != 3 {
		errors.GenerateError(t, fmt.Sprintf("The url %s with headers %v should be candidate for %v tags, %v given", baseUrl, headers, []string{FIRST_KEY, THIRD_KEY, FOURTH_KEY}, r.GetValidatedTags(baseUrl, nil)))
	}
	if len(r.GetValidatedTags(validUrl, headers)) != 4 {
		errors.GenerateError(t, fmt.Sprintf("The url %s with headers %v should be candidate for %v tags, %v given", baseUrl, headers, []string{FIRST_KEY, SECOND_KEY, THIRD_KEY, FOURTH_KEY}, r.GetValidatedTags(baseUrl, nil)))
	}
}
