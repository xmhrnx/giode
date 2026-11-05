// Package lazylist provides a vertical list of rows of any height that
// only builds and lays out the rows in or near its viewport - giode's
// FlatList, ListView.builder or LazyColumn.

package lazylist

func reverse(p []placed) {
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
}
