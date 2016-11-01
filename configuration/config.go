package configuration

// Config contains the settings for alpr and mqtt
type Config struct {
	Alpr AlprConfig `yaml:"alpr"`
	MQTT MQTTConfig `yaml:"mqtt"`
}

// AlprConfig contains the settings used to start and use the ALPR software
type AlprConfig struct {
	Location   string  `yaml:"location"`   // location of alpr software (including executable of some sort)
	Stream     string  `yaml:"stream"`     // url to the video stream
	Confidence float64 `yaml:"confidence"` // confidence 0-100, plate will not be used if confidence is below this value
	Lost       int64   `yaml:"lost"`       // set time in ms after what time the program considers a plate as lost, a published plate will be kept in memory for the time defined by lost and will not be published again
	ScanTime   int64   `yaml:"scanTime"`   // set time in ms after what time the program should publish the number plate after first recognition, higher scan time can mean higher confidence rate. 3000 should be ok. currently not fully implemented
}

// MQTTConfig contains the MQTT client information
type MQTTConfig struct {
	Host     string `yaml:"host"`     // host of the mqtt broker
	Port     int    `yaml:"port"`     // port of the mqtt broker
	StreamID int    `yaml:"streamId"` // stream id of sensorthings datastream
	ClientID string `yaml:"clientId"` // mqtt client id to use
}
