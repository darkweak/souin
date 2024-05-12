package templates

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/template"
)

// ValidateTemplate validates a text template results in valid JSON
// when it's executed with empty template data. If template execution
// results in invalid JSON, the template is invalid. When the template
// is valid, it can be used safely. A valid template can still result
// in invalid JSON when non-empty template data is provided.
func ValidateTemplate(data []byte, funcMap template.FuncMap) error {
	if len(data) == 0 {
		return nil
	}

	// prepare the template with our template functions
	_, err := template.New("template").Funcs(funcMap).Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	return nil
}

// ValidateTemplateData validates that template data is
// valid JSON.
func ValidateTemplateData(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if ok := json.Valid(data); !ok {
		return errors.New("error validating json template data")
	}

	return nil
}
