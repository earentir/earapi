package health

import (
	"fmt"
	"strings"
)

// FormatFoodsCreatedReply is returned on POST /log and POST /fill when new catalog rows were inserted.
func FormatFoodsCreatedReply(created []Food) string {
	if len(created) == 0 {
		return ""
	}
	names := make([]string, 0, len(created))
	for _, f := range created {
		n := strings.TrimSpace(f.Name)
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"New catalog food(s) added: %s. Existing foods are matched by fuzzy search first; these names did not match anything in the catalog. Review the Foods tab in the web UI and merge or delete duplicates if needed.",
		strings.Join(names, ", "),
	)
}
