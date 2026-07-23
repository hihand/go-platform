package configkit

// configFileType is the Viper config-type marker set on
// loadFile. Viper needs an explicit type when SetConfigFile is used
// because the extension alone is ambiguous once the path comes from
// user configuration.
//
// Centralised here so the option, the loader, and the test harness
// all agree on the same value with no string drift.
const configFileType = "yaml"
