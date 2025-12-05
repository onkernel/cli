package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplates(t *testing.T) {
	// Should have at least one template
	assert.NotEmpty(t, Templates, "Templates map should not be empty")

	// Sample app should exist
	sampleApp, exists := Templates["sample-app"]
	assert.True(t, exists, "sample-app template should exist")

	// Sample app should have required fields
	assert.NotEmpty(t, sampleApp.Name, "Template should have a name")
	assert.NotEmpty(t, sampleApp.Description, "Template should have a description")
	assert.NotEmpty(t, sampleApp.Languages, "Template should support at least one language")

	// Should support both typescript and python
	assert.Contains(t, sampleApp.Languages, string(LanguageTypeScript), "sample-app should support typescript")
	assert.Contains(t, sampleApp.Languages, string(LanguagePython), "sample-app should support python")
}
