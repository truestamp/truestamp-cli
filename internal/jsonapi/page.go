// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package jsonapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// SetPageQuery writes the page parameters every keyset-paged list shares:
// page[limit] for the page size, page[after] or page[before] to continue
// from a cursor in either direction, and page[count]=true when the caller
// wants the server's total. Sort is the caller's, and must be re-sent on
// every page: this client rebuilds the query rather than following the
// server's links, so nothing carries over.
func SetPageQuery(q url.Values, limit int, after, before string, count bool) {
	q.Set("page[limit]", strconv.Itoa(limit))
	if after != "" {
		q.Set("page[after]", after)
	}
	if before != "" {
		q.Set("page[before]", before)
	}
	if count {
		q.Set("page[count]", "true")
	}
}

// SortByID is the one ordering every list uses: by id, which for the
// ULIDs and UUIDv7s these resources carry is insertion time. Newest first
// unless the caller asked to start from the beginning.
func SortByID(oldestFirst bool) string {
	if oldestFirst {
		return "id"
	}
	return "-id"
}

// PageInfo is what a list response says about the pages around it.
type PageInfo struct {
	// NextCursor is the page[after] value lifted from links.next; empty on
	// the last page. An unparseable link also reads as empty, which is the
	// safe reading: a bad cursor would otherwise loop.
	NextCursor string
	// PrevCursor is the page[before] value lifted from links.prev; empty on
	// the first page.
	PrevCursor string
	// Total is meta.page.total, present only when the request asked for
	// page[count]=true; zero otherwise.
	Total int
	// Limit is meta.page.limit, the page size the server actually used.
	// Every collection clamps page[limit] to its max_page_size (250 by
	// default) rather than refusing it, so this can be smaller than what
	// was asked for; zero when the server did not report one.
	Limit int
}

// ParsePage reads links.next, links.prev and meta.page.total from a list
// body. It tolerates any of them being absent.
func ParsePage(body []byte) PageInfo {
	var env struct {
		Links struct {
			Next string `json:"next"`
			Prev string `json:"prev"`
		} `json:"links"`
		Meta struct {
			Page struct {
				Total int `json:"total"`
				Limit int `json:"limit"`
			} `json:"page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return PageInfo{}
	}
	return PageInfo{
		NextCursor: cursorFromLink(env.Links.Next, "page[after]"),
		PrevCursor: cursorFromLink(env.Links.Prev, "page[before]"),
		Total:      env.Meta.Page.Total,
		Limit:      env.Meta.Page.Limit,
	}
}

func cursorFromLink(link, param string) string {
	if link == "" {
		return ""
	}
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Query().Get(param)
}
