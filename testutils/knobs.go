package testutils

type FetchTestingKnobs struct {
	// Used to simulate testing when the CSV input file is wrong.
	TriggerCorruptCSVFile bool
}
