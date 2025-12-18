package calm

import (
	"net/url"
	"sort"
	"strings"
)

type UrlBuilder struct {
	u     *url.URL
	query url.Values
}

func ParseUrl(rawUrl string) *UrlBuilder {
	ub := &UrlBuilder{}
	ub.u, _ = url.Parse(rawUrl)
	ub.query = ub.u.Query()
	return ub
}

func (builder *UrlBuilder) AddQuery(name, value string) *UrlBuilder {
	builder.query.Add(name, value)
	return builder
}

func (builder *UrlBuilder) AddQueries(queries map[string]string) *UrlBuilder {
	for name, value := range queries {
		builder.AddQuery(name, value)
	}
	return builder
}

func (builder *UrlBuilder) GetQuery() url.Values {
	return builder.query
}

func (builder *UrlBuilder) GetRawQuery() string {
	return builder.u.RawQuery
}

func (builder *UrlBuilder) GetURL() *url.URL {
	return builder.u
}

func (builder *UrlBuilder) Build() *url.URL {
	builder.u.RawQuery = builder.query.Encode()
	return builder.u
}

func (builder *UrlBuilder) BuildStr() string {
	return builder.Build().String()
}

func (builder *UrlBuilder) BuildString() string {
	if len(builder.query) == 0 {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(builder.query))
	for k := range builder.query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := builder.query[k]
		for _, v := range vs {

			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}

	builder.u.RawQuery = buf.String()

	return builder.u.String()
}
