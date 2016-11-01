package models

type Response struct {
	Version        float32  `json:"version"`
	DataType       string   `json:"data_type"`
	EpochTime      float64  `json:"epoch_time"`
	ImgWidth       int      `json:"img_width"`
	ImgHeight      int      `json:"img_height"`
	ProcessingTime float64  `json:"processing_time_ms"`
	Results        []Result `json:"results"`
}

type Result struct {
	Plate           string  `json:"plate"`
	Confidence      float64 `json:"confidence"`
	MatchesTemplate int     `json:"matches_template"`
	PlateIndex      int     `json:"plate_index"`
	Region          string  `json:"region"`
	FirstSeen       int64   `json:"firstSeen"`
	LastSeen        int64   `json:"lastSeen"`
}

// MQTTClient interface defines the needed MQTT client operations
type MQTTClient interface {
	Start()
	Stop()
	Publish(string, string, byte) //topic, message, qos
}
