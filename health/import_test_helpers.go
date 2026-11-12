package health

// aliasFoodsFromMDDir loads Food aliases.md for import scaling tests.
func aliasFoodsFromMDDir(dir string, foods []Food) map[string]Food {
	specs, _ := LoadFoodAliasesFromMDDir(dir)
	return BuildAliasFoodMap(specs, foods)
}
