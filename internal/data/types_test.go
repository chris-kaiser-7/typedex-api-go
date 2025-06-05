//go:build openai

package data

import (
	"golang.org/x/exp/slices"
	"testing"
)

func TestGetSubtypesById(t *testing.T) {
	tests := []struct {
		name      string
		ids       []int64
		want      []string
		expectErr bool
	}{
		{"Valid IDs", subtypeTestIds, []string{"Orc", "Goblin"}, false},
		{"Invalid ID (zero)", []int64{0}, nil, true},
		{"Nonexistent ID", []int64{999}, []string{}, false},
		{"empty ids", []int64{}, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := subtypeDa.getSubtypesById(tt.ids)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !slices.Equal(got, tt.want) && !tt.expectErr {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// this test kinda sucks and is kinda pointless...
func TestSubtypeDA_Get(t *testing.T) {
	for index, subtypeId := range subtypeTestIds {
		subtype, err := subtypeDa.Get(int64(subtypeId))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assertion
		if subtype.ID != int64(subtypeId) {
			t.Errorf("expected subtype ID %d, got %d", subtypeId, subtype.ID)
		}
		if subtype.Book.ID != bookTestId {
			t.Errorf("expected book ID %d, got %d", bookTestId, subtype.Book.ID)
		}
		if subtype.Book.Name != "Test Book" {
			t.Errorf("expected book name 'Test Book', got '%s'", subtype.Book.Name)
		}
		//TODO: replace magic test strings with test sets
		if index == 0 {
			if subtype.TypeName != "Orc" {
				t.Errorf("expected type name 'Orc', got '%s'", subtype.TypeName)
			}
			if subtype.PropValues[0] != "Orc" || subtype.PropValues[1] != "200" {
				t.Errorf("unexpected properties: %v", subtype.PropValues)
			}
			if len(subtype.Children) == 0 {
				t.Errorf("expected children length > 0, got %v", subtype.Children)
			}
			if subtype.Children[0] != subtypeTestIds[1] {
				t.Errorf("expected child %v, got %v", subtypeTestIds[0], subtype.Children[0])
			}
		} else {
			if subtype.TypeName != "Goblin" {
				t.Errorf("expected type name 'Goblin', got '%s'", subtype.TypeName)
			}
			if subtype.PropValues[0] != "Goblin" || subtype.PropValues[1] != "100" {
				t.Errorf("unexpected properties: %v", subtype.PropValues)
			}
			if *subtype.Parent != subtypeTestIds[0] {
				t.Errorf("expected parent %v, got %v", subtypeTestIds[0], *subtype.Parent)
			}
		}
	}
}

func TestSubtypeDA_GenerateChildren(t *testing.T) {
	tests := []struct {
		name          string
		parentID      int64
		expectedCount int
	}{
		{"Generate 10 children with id 0", subtypeTestIds[0], 10},
		{"Generate 3 children with id 1", subtypeTestIds[1], 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, err := subtypeDa.Get(tt.parentID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			children, err := subtypeDa.generateChildren(&parent, tt.expectedCount)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(children) != tt.expectedCount {
				t.Fatalf("incorrect child count: got %v expected %v", len(children), tt.expectedCount)
			}

			for _, child := range children {
				assertMatch(*child.Parent, parent.ID, "child with incorrect parent id.", t)
				assertMatch(len(child.Ancestry), len(parent.Ancestry)+1, "child has incorrect ancestry length.", t)
				assertMatch(child.Book.ID, parent.Book.ID, "child has incorrect book ID.", t)
				assertMatch(len(child.PropValues), len(parent.Book.Fields), "child has incorrect prop values count.", t)
				assertDontMatch(child.TypeName, "", "child has empty string type name", t)

				if !slices.Contains(parent.Children, child.ID) {
					t.Fatalf("Child is not in parents children: parent %v child %v", parent.Children, child.ID)
				}
			}
		})
	}
}

// func TestSubtypeDA_GenerateChildren(t *testing.T) {
// 	parent, err := subtypeDa.Get(subtypeTestIds[1])
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	children, err := subtypeDa.generateChildren(&parent, 3)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if len(children) != 3 {
// 		t.Fatalf("incorrect child count: got %v expected 3", len(children))
// 	}
// 	for _, child := range children {
// 		assertMatch(*child.Parent, parent.ID, "child with incorrect parent id.", t)
// 		assertMatch(len(child.Ancestry), len(parent.Ancestry)+1, "child has incorrect ancestry length.", t)
// 		assertMatch(child.Book.ID, parent.Book.ID, "child has incorrect book ID.", t)
// 		assertMatch(len(child.PropValues), len(parent.Book.Fields), "child has incorrect prop values count.", t)
// 		assertDontMatch(child.TypeName, "", "child has empty string type name", t)
//
// 		if !slices.Contains(parent.Children, child.ID) {
// 			t.Fatalf("Child is not in parents children: parent %v child %v", parent.Children, child.ID)
// 		}
// 	}
// }
//
// func TestSubtypeDA_GenerateChildren2(t *testing.T) {
// 	parent, err := subtypeDa.Get(subtypeTestIds[0])
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	children, err := subtypeDa.generateChildren(&parent, 10)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if len(children) != 10 {
// 		t.Fatalf("incorrect child count: got %v expected 10", len(children))
// 	}
// 	for _, child := range children {
// 		assertMatch(*child.Parent, parent.ID, "child with incorrect parent id.", t)
// 		assertMatch(len(child.Ancestry), len(parent.Ancestry)+1, "child has incorrect ancestry length.", t)
// 		assertMatch(child.Book.ID, parent.Book.ID, "child has incorrect book ID.", t)
// 		assertMatch(len(child.PropValues), len(parent.Book.Fields), "child has incorrect prop values count.", t)
// 		assertDontMatch(child.TypeName, "", "child has empty string type name", t)
//
// 		if !slices.Contains(parent.Children, child.ID) {
// 			t.Fatalf("Child is not in parents children: parent %v child %v", parent.Children, child.ID)
// 		}
// 	}
// }

func assertMatch[T comparable](got T, match T, mesg string, t *testing.T) {
	if got != match {
		t.Fatalf("%s got %v expected %v.", mesg, got, match)
	}
}

func assertDontMatch[T comparable](got T, match T, mesg string, t *testing.T) {
	if got == match {
		t.Fatalf("%s got %v", mesg, got)
	}
}
