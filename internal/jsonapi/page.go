// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package jsonapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// SetPageQuery writes the page parameters every keyset-paged list shares:
// page[limit] for the page size, page[after] to continue from a cursor,
// and page[count]=true when the caller wants the server's total. Sort is
// the caller's, and must be re-sent on every page: this client rebuilds
// the query rather than following links.next, so nothing carries over.
func SetPageQuery(q url.Values, limit int, after string, count bool) {
	q.Set("page[limit]", strconv.Itoa(limit))
	if after != "" {
		q.Set("page[after]", after)
	}
	if count {
		q.Set("page[count]", "true")
	}
}

// PageInfo is what a list response says about the pages around it.
type PageInfo struct {
	// NextCursor is the page[after] value lifted from links.next; empty on
	// the last page. An unparseable link also reads as empty, which is the
	// safe reading: a bad cursor would otherwise loop.
	NextCursor string
	// Total is meta.page.total, present only when the request asked for
	// page[count]=true; zero otherwise.
	Total int
}

// ParsePage reads links.next and meta.page.total from a list body. It
// tolerates either being absent.
func ParsePage(body []byte) PageInfo {
	var env struct {
		Links struct {
			Next string `json:"next"`
		} `json:"links"`
		Meta struct {
			Page struct {
				Total int `json:"total"`
			} `json:"page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return PageInfo{}
	}
	return PageInfo{NextCursor: cursorFromNextLink(env.Links.Next), Total: env.Meta.Page.Total}
}

func cursorFromNextLink(next string) string {
	if next == "" {
		return ""
	}
	u, err := url.Parse(next)
	if err != nil {
		return ""
	}
	return u.Query().Get("page[after]")
}
