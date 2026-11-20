package domain

import "testing"

func TestIsValidContentSort(t *testing.T) {
	valid := []ContentSort{
		ContentSortUpdatedDesc,
		ContentSortUpdatedAsc,
		ContentSortCreatedDesc,
		ContentSortCreatedAsc,
		ContentSortTitleAsc,
		ContentSortTitleDesc,
	}
	for _, s := range valid {
		if !IsValidContentSort(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []ContentSort{
		"",
		"updated",
		"title",
		"UPDATED_DESC",
		"invalid_sort",
	}
	for _, s := range invalid {
		if IsValidContentSort(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
