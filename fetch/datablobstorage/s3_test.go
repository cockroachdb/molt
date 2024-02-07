package datablobstorage

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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

type mockS3Client struct {
	s3iface.S3API
	listResp *s3.ListObjectsV2Output
	getResp  *s3.GetObjectOutput
	err      error
}

func (m *mockS3Client) ListObjectsV2WithContext(
	ctx context.Context, params *s3.ListObjectsV2Input, opts ...request.Option,
) (*s3.ListObjectsV2Output, error) {
	return m.listResp, m.err
}

func (m *mockS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return m.getResp, m.err
}

func TestListFromContinuationPointAWS(t *testing.T) {
	rows := "10"
	s3CLI := &mockS3Client{
		listResp: &s3.ListObjectsV2Output{
			Contents: []*s3.Object{
				{Key: aws.String("part_00000001.tar.gz")},
				{Key: aws.String("part_00000002.tar.gz")},
				{Key: aws.String("part_00000003.tar.gz")},
				{Key: aws.String("part_00000004.tar.gz")},
				{Key: aws.String("part_00000005.tar.gz")},
				{Key: aws.String("part_00000006.tar.gz")},
				{Key: aws.String("part_00000007.tar.gz")},
				{Key: aws.String("part_00000008.tar.gz")},
			},
		},
		getResp: &s3.GetObjectOutput{
			Metadata: map[string]*string{
				numRowKeysAWS: &rows,
			},
		},
		err: nil,
	}
	ctx := context.Background()
	s3Store := s3Store{
		bucket: "fetch-test",
		creds: credentials.Value{
			AccessKeyID:     "aaaa",
			SecretAccessKey: "bbbb",
		},
	}
	resources, err := listFromContinuationPointAWS(ctx, s3CLI, "part_00000004.tar.gz", "public.inventory", &s3Store)
	require.NoError(t, err)
	require.Len(t, resources, 5)
}
