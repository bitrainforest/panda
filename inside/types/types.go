package types

type Sector struct {
	ID int
	// we max retry three times
	Try int
}
