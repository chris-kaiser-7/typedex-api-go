package data

import (
	"testing"
)

func TestPromptTemplateModel_InsertAndGet(t *testing.T) {
	// Create template
	template := &PromptTemplate{
		Name:     "Test Template",
		Template: "Prompt: %s",
	}

	err := promptDa.Insert(template)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if template.ID == 0 {
		t.Fatal("expected template ID to be set")
	}

	// Get template
	got, err := promptDa.Get(template.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != template.Name {
		t.Errorf("expected name %q, got %q", template.Name, got.Name)
	}
	if got.Template != template.Template {
		t.Errorf("expected template %q, got %q", template.Template, got.Template)
	}
}
