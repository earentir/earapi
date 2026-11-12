package health

import (
	"fmt"
	"os"
)

// ImportMarkdownForUser loads mddata into the shared DB for one user (replace).
func ImportMarkdownForUser(dataRoot, mdDir, username string) (map[string]int, error) {
	fmt.Fprintf(os.Stderr, "health: opening %s …\n", dataRoot)
	db, err := OpenSharedDB(dataRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	st, err := NewStore(db, dataRoot, username)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "health: parsing %s …\n", mdDir)
	w, f, d, stock, err := ImportFromMDDir(mdDir)
	if err != nil {
		return nil, err
	}
	if len(w) == 0 && len(f) == 0 && len(d) == 0 && len(stock) == 0 {
		return nil, fmt.Errorf("no data found in %s", mdDir)
	}
	fmt.Fprintf(os.Stderr, "health: writing DB (%d days, %d stock lots) …\n", len(d), len(stock))
	if err := st.ReplaceImported(w, f, d, stock); err != nil {
		return nil, err
	}
	return map[string]int{
		"weights": len(w),
		"foods":   len(f),
		"days":    len(d),
		"stock":   len(stock),
	}, nil
}
