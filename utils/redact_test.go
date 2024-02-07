package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeExternalStorageURI(t *testing.T) {
	testCases := []struct {
		name             string
		inputURI         string
		inputExtraParams []string
		expected         string
	}{
		{
			name:     "redacts password",
			inputURI: "http://username:password@foo.com/something",
			expected: "http://username:redacted@foo.com/something",
		},
		{
			name:             "redacts given parameters",
			inputURI:         "http://foo.com/something?secret_key=uhoh",
			inputExtraParams: []string{"secret_key"},
			expected:         "http://foo.com/something?secret_key=redacted",
		},
		{
			name:     "redacts gcp",
			inputURI: "gs://migrations-fetch-ci-test/fetch/public.inventory/part_00000002.tar.gz?CREDENTIALS=abvdec",
			expected: "gs://migrations-fetch-ci-test/fetch/public.inventory/part_00000002.tar.gz?CREDENTIALS=redacted",
		},
		{
			name:     "redacts s3",
			inputURI: "s3://test-data/fetch/public.inventory/part_00000001.tar.gz?AWS_ACCESS_KEY_ID=ABSDERDFG&AWS_SECRET_ACCESS_KEY=qwertdfgd",
			expected: "s3://test-data/fetch/public.inventory/part_00000001.tar.gz?AWS_ACCESS_KEY_ID=ABSDERDFG&AWS_SECRET_ACCESS_KEY=redacted",
		},
	}
	RedactedQueryParams = map[string]struct{}{AWSSecretAccessKey: {}, GCPCredentials: {}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualOutput, err := SanitizeExternalStorageURI(tc.inputURI, tc.inputExtraParams)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actualOutput)
		})
	}
}
