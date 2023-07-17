package server

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskSecretURLParams(t *testing.T) {
	secretKeywords := []string{"secret", "token", "key", "password"}
	mask := "MASKED"

	type testCase struct {
		name     string
		rawURL   string
		expected string
	}

	cases := []testCase{
		{
			name:     "no params",
			rawURL:   "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "user info redacted",
			rawURL:   "https://user:P@$$w0rD@example.com/foo/bar?baz=qux&secret_key=baz",
			expected: "https://user:xxxxx@example.com/foo/bar?baz=qux&secret_key=" + mask,
		},
	}
	for i, kw := range secretKeywords {
		cases = append(cases, testCase{
			name:     "single " + kw,
			rawURL:   "https://example.com?" + kw + "=foo",
			expected: "https://example.com?" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw,
			rawURL:   "https://example.com?" + kw + "=foo" + "&bar_" + kw + "=bar",
			expected: "https://example.com?" + kw + "=" + mask + "&bar_" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw + " with other params",
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw + "=bar",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw + "=" + mask,
		})
		cases = append(cases, testCase{
			name:     "multiple " + kw + " with other params and fragment",
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw + "=bar#fragment",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw + "=" + mask + "#fragment",
		})
		kw2 := secretKeywords[(i+1)%len(secretKeywords)]
		cases = append(cases, testCase{
			name:     "combined " + kw + " and " + kw2,
			rawURL:   "https://example.com?foo=bar&" + kw + "=foo" + "&bar_" + kw2 + "=bar",
			expected: "https://example.com?foo=bar&" + kw + "=" + mask + "&bar_" + kw2 + "=" + mask,
		})
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			masked := maskSecretURLParams(c.rawURL)
			got, err := url.Parse(masked)
			require.NoError(t, err)
			want, err := url.Parse(c.expected)
			require.NoError(t, err)
			require.EqualValues(t, got.Query(), want.Query())
			require.Equal(t, got.Fragment, want.Fragment)
			require.Equal(t, got.Host, want.Host)
			require.Equal(t, got.Path, want.Path)
			require.Equal(t, got.Scheme, want.Scheme)
		})
	}
}