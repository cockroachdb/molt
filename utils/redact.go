package utils

import "net/url"

const (
	redactionMarker = "redacted"
)

// redactedQueryParams is the set of query parameter names registered by the
// external storage providers that should be redacted from external storage URIs
// whenever they are displayed to a user.
var RedactedQueryParams = map[string]struct{}{}

// SanitizeExternalStorageURI returns the external storage URI with with some
// secrets redacted, for use when showing these URIs in the UI, to provide some
// protection from shoulder-surfing. The param is still present -- just
// redacted -- to make it clearer that that value is indeed persisted interally.
// extraParams which should be scrubbed -- for params beyond those that the
// various cloud-storage URIs supported by this package know about -- can be
// passed allowing this function to be used to scrub other URIs too (such as
// non-cloudstorage changefeed sinks).
func SanitizeExternalStorageURI(path string, extraParams []string) (string, error) {
	uri, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	if uri.User != nil {
		if _, passwordSet := uri.User.Password(); passwordSet {
			uri.User = url.UserPassword(uri.User.Username(), redactionMarker)
		}
	}

	params := uri.Query()
	for param := range params {
		if _, ok := RedactedQueryParams[param]; ok {
			params.Set(param, redactionMarker)
		} else {
			for _, p := range extraParams {
				if param == p {
					params.Set(param, redactionMarker)
				}
			}
		}
	}

	uri.RawQuery = params.Encode()
	return uri.String(), nil
}
