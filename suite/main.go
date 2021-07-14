package suite

var (
	defaultConfigTestPath        = "../suite/config_test.yaml"
	defaultFileSdTestInitialPath = "../suite/file_sd_test.json"
	defaultFileSdTestModPath     = "../suite/file_sd_test_2.json"
)

func GetConfigTestFile() string {
	return defaultConfigTestPath
}

func GetFileSdTestInitialFile() string {
	return defaultFileSdTestInitialPath
}

func GetFileSdTestModFile() string {
	return defaultFileSdTestModPath
}
