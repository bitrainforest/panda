package types

type SectorDownloadStatus int

const (
	// 1111
	NeedFour SectorDownloadStatus = 15
	// 1110
	NeedThree   SectorDownloadStatus = 14
	NeedTwo     SectorDownloadStatus = 12
	NeedOne     SectorDownloadStatus = 8
	NeedNothing SectorDownloadStatus = 0

	// 0001
	DetectNeedDownloadSealed = 1
	// 0010
	DetectNeedDownloadCache = 2
	// 0100
	DetectNeedDeclare = 4
	// 1000
	DetectNeedCallback = 8

	NeedDownloadSealed = 1
	NeedDownloadCache  = 2
	NeedDeclare        = 4
	NeedCallback       = 8
)

type Sector struct {
	ID int
	// we max retry three times
	Try int
	/*
		    1111: means do all operate, download sealed, cache, declare, callback
			1110: means do cache, declare, callback
			1100: means declare, callback
			1000: means just do callback
	*/
	Status SectorDownloadStatus
}

func (s Sector) NeedDownloadSealed() bool {
	return (s.Status & DetectNeedDownloadSealed) == NeedDownloadSealed
}

func (s Sector) NeedDownloadCache() bool {
	return (s.Status & DetectNeedDownloadCache) == NeedDownloadCache
}

func (s Sector) NeedDeclare() bool {
	return (s.Status & DetectNeedDeclare) == NeedDeclare
}

func (s Sector) NeedCallback() bool {
	return (s.Status & DetectNeedCallback) == NeedCallback
}
