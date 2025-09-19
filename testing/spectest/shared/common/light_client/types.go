package light_client

type Step struct {
	NewBlock *DataEntry `json:"new_block"`
	NewHead  *HeadEntry `json:"new_head"`
}

type DataEntry struct {
	ForkDigest string `json:"fork_digest"`
	Data       string `json:"data"`
}

type HeadEntry struct {
	HeadBlockRoot string       `json:"head_block_root"`
	Checks        *ChecksEntry `json:"checks"`
}

type ChecksEntry struct {
	LatestFinalizedCheckpoint EpochRoot           `json:"latest_finalized_checkpoint"`
	LatestFinalityUpdate      DataEntry           `json:"latest_finality_update"`
	LatestOptimisticUpdate    DataEntry           `json:"latest_optimistic_update"`
	Bootstraps                []*BootstrapEntry   `json:"bootstraps"`
	BestUpdates               []*BestUpdatesEntry `json:"best_updates"`
}

type BestUpdatesEntry struct {
	Period int       `json:"period"`
	Update DataEntry `json:"update"`
}

type BootstrapEntry struct {
	BlockRoot string    `json:"block_root"`
	Bootstrap DataEntry `json:"bootstrap"`
}

type EpochRoot struct {
	Epoch int    `json:"epoch"`
	Root  string `json:"root"`
}
