package datablobstorage

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/require"
)

func TestS3Resource_ImportURL(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		r        *s3Resource
		expected string
	}{
		{
			desc: "basic",
			r: &s3Resource{
				key: "asdf/ghjk.csv",
				store: &s3Store{
					bucket: "nangs",
					creds: credentials.Value{
						AccessKeyID:     "aaaa",
						SecretAccessKey: "bbbb",
					},
				},
			},
			expected: "s3://nangs/asdf/ghjk.csv?AWS_ACCESS_KEY_ID=aaaa&AWS_SECRET_ACCESS_KEY=bbbb",
		},
		{
			desc: "url escaped",
			r: &s3Resource{
				key: "asdf/ghjk.csv",
				store: &s3Store{
					bucket: "nangs",
					creds: credentials.Value{
						AccessKeyID:     "aaa a",
						SecretAccessKey: "b&bbb",
					},
				},
			},
			expected: "s3://nangs/asdf/ghjk.csv?AWS_ACCESS_KEY_ID=aaa+a&AWS_SECRET_ACCESS_KEY=b%26bbb",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := tc.r.ImportURL()
			require.NoError(t, err)
			require.Equal(t, tc.expected, u)
		})
	}
}
