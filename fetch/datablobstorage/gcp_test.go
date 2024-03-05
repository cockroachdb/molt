package datablobstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
)

func TestGCPResource_ImportURL(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		r        *gcpResource
		expected string
	}{
		{
			desc: "basic test",
			r: &gcpResource{
				key: "asdf/ghjk.csv",
				store: &gcpStore{
					bucket: "nangs",
					creds: &google.Credentials{
						JSON: []byte(`{"a":b}`),
					},
				},
			},
			expected: "gs://nangs/asdf/ghjk.csv?CREDENTIALS=eyJhIjpifQ==",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := tc.r.ImportURL()
			require.NoError(t, err)
			require.Equal(t, tc.expected, u)
		})
	}
}
