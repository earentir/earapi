package health

// EU Reg. 1169/2011: declared “salt” is NaCl (g); sodium is 39.34% of NaCl by mass.
const SodiumMgPerGramSalt = 393.34

// SodiumMgFromSaltPer100g converts label salt (g per 100 g) to sodium (mg per 100 g).
func SodiumMgFromSaltPer100g(saltG float64) float64 {
	return saltG * SodiumMgPerGramSalt
}

// SaltGFromSodiumMgPer100g converts sodium (mg per 100 g) to label salt (g per 100 g).
func SaltGFromSodiumMgPer100g(sodiumMg float64) float64 {
	if sodiumMg <= 0 {
		return 0
	}
	return sodiumMg / SodiumMgPerGramSalt
}