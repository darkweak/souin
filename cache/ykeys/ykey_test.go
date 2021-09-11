package ykeys

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/plugins/souin/configuration"
)

const FirstKey = "The_First_Test"
const SecondKey = "The_Second_Test"
const ThirdKey = "The_Third_Test"
const FourthKey = "The_Fourth_Test"

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

	if nil == r || nil == r.Keys {
		errors.GenerateError(t, "The key saver should be instanciated")
	}
}

func TestYKeyStorage_AddToTags(t *testing.T) {
	baseYkeys := mockYkeys()
	url := "http://the.url.com"

	r := InitializeYKeys(baseYkeys)
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url, []string{FirstKey})
	time.Sleep(200 * time.Millisecond)
	res, _ := r.Cache.Get(FirstKey)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R1 => The key %s should contains the url %s, %s given", FirstKey, url, res))
	}

	r.AddToTags(url, []string{FirstKey, SecondKey})
	r.AddToTags("http://domain.com", []string{FirstKey})

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(FirstKey)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains the url %s, %s given", FirstKey, url, res))
	}
	if len(strings.Split(res.(string), ",")) != 2 {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains 2 records, %d given", FirstKey, len(strings.Split(res.(string), ","))))
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(SecondKey)
	if !strings.Contains(res.(string), url) {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains the url %s, %s given", SecondKey, url, res))
	}
	if len(strings.Split(res.(string), ",")) != 1 {
		errors.GenerateError(t, fmt.Sprintf("R2 => The key %s should contains 2 records, %d given", FirstKey, len(strings.Split(res.(string), ","))))
	}

	r.AddToTags(url, []string{FirstKey})
	r.AddToTags(url, []string{FirstKey})
	r.AddToTags("http://domain.com", []string{FirstKey})
	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(FirstKey)
	if len(strings.Split(res.(string), ",")) != 2 {
		errors.GenerateError(t, fmt.Sprintf("R3 => The key %s should contains 2 records, %d given", FirstKey, len(strings.Split(res.(string), ","))))
	}
}

func TestYKeyStorage_InvalidateTags(t *testing.T) {
	baseYkeys := mockYkeys()
	url1 := "http://the.url1.com"
	url2 := "http://the.url2.com"
	url3 := "http://the.url3.com"
	url4 := "http://the.url4.com"

	r := InitializeYKeys(baseYkeys)
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url1, []string{FirstKey, SecondKey})
	r.AddToTags(url2, []string{FirstKey, SecondKey})
	r.AddToTags(url3, []string{FirstKey, ThirdKey})
	r.AddToTags(url4, []string{ThirdKey, FourthKey})

	urls := r.InvalidateTags([]string{FirstKey})

	if len(urls) != 3 {
		errors.GenerateError(t, fmt.Sprintf("It should have 3 urls to remove (e.g. [http://the.url1.com http://the.url2.com http://the.url3.com]), %v given", urls))
	}

	time.Sleep(200 * time.Millisecond)
	res, _ := r.Cache.Get(FirstKey)
	if res.(string) != "" {
		errors.GenerateError(t, "The FIRST_KEY should be empty")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(SecondKey)
	if res.(string) != "" {
		errors.GenerateError(t, "The SECOND_KEY should be empty")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(ThirdKey)
	if res.(string) != url4 {
		errors.GenerateError(t, "The THIRD_KEY should be equals to http://the.url4.com")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(FourthKey)
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
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url1, []string{FirstKey, SecondKey})
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url2, []string{FirstKey, SecondKey})
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url3, []string{FirstKey, ThirdKey})
	time.Sleep(200 * time.Millisecond)
	r.AddToTags(url4, []string{ThirdKey, FourthKey})

	urls := r.InvalidateTagURLs(fmt.Sprintf("%s,%s", url1, url3))
	if len(urls) != 2 {
		errors.GenerateError(t, "It should have 2 urls to remove (e.g. [http://the.url1.com http://the.url3.com])")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ := r.Cache.Get(FirstKey)
	if res.(string) != url2 {
		errors.GenerateError(t, "The FIRST_KEY should be equals to http://the.url2.com")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(SecondKey)
	if res.(string) != url2 {
		errors.GenerateError(t, "The SECOND_KEY should be equals to http://the.url2.com")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(ThirdKey)
	if res.(string) != url4 {
		errors.GenerateError(t, "The THIRD_KEY should be equals to http://the.url4.com")
	}

	time.Sleep(200 * time.Millisecond)
	res, _ = r.Cache.Get(FourthKey)
	if res.(string) != url4 {
		errors.GenerateError(t, "The FOURTH_KEY should be equals to http://the.url4.com")
	}
}

func TestYKeyStorage_GetValidatedTags(t *testing.T) {
	baseYkeys := mockYkeys()
	r := InitializeYKeys(baseYkeys)
	baseURL := "http://the.url.com"
	invalidURL := "http://domain.com/test/the/second"
	validURL := invalidURL + "/anything"
	if len(r.GetValidatedTags(baseURL, nil)) != 2 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseURL, []string{ThirdKey, FourthKey}, r.GetValidatedTags(baseURL, nil)))
	}
	if len(r.GetValidatedTags(invalidURL, nil)) != 2 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseURL, []string{ThirdKey, FourthKey}, r.GetValidatedTags(baseURL, nil)))
	}
	if len(r.GetValidatedTags(validURL, nil)) != 3 {
		errors.GenerateError(t, fmt.Sprintf("The url %s without headers should be candidate for %v tags, %v given", baseURL, []string{SecondKey, ThirdKey, FourthKey}, r.GetValidatedTags(baseURL, nil)))
	}

	headers := http.Header{}
	headers.Set("Authorization", "anything")
	headers.Set("Content-Type", "any value here")
	if len(r.GetValidatedTags(baseURL, headers)) != 3 {
		errors.GenerateError(t, fmt.Sprintf("The url %s with headers %v should be candidate for %v tags, %v given", baseURL, headers, []string{FirstKey, ThirdKey, FourthKey}, r.GetValidatedTags(baseURL, nil)))
	}
	if len(r.GetValidatedTags(validURL, headers)) != 4 {
		errors.GenerateError(t, fmt.Sprintf("The url %s with headers %v should be candidate for %v tags, %v given", validURL, headers, []string{FirstKey, SecondKey, ThirdKey, FourthKey}, r.GetValidatedTags(validURL, nil)))
	}
}
