package database

import (
	"log"
	"os"
	"strconv"

	"github.com/kkrypt0nn/spaceflake"
)

var settings = initSnowflake()

// initSnowflake initializes the snowflake generator
func initSnowflake() spaceflake.GeneratorSettings {
	settings := spaceflake.NewGeneratorSettings()
	settings.BaseEpoch = 1706639400000 // January 30, 2024 12:30:00 PM Central/Regina
	nodeID, err := strconv.ParseUint(os.Getenv("SNOWFLAKE_NODE_ID"), 10, 64)
	if err != nil {
		nodeID = 1
		log.Println("Failed to parse SNOWFLAKE_NODE_ID, defaulting to 1")
	}
	settings.NodeID = nodeID
	workID, _ := strconv.ParseUint(os.Getenv("SNOWFLAKE_WORKER_ID"), 10, 64)
	if err != nil {
		workID = 1
		log.Println("Failed to parse SNOWFLAKE_WORKER_ID, defaulting to 1")
	}
	settings.WorkerID = workID
	settings.Sequence = 0
	return settings
}

// GenSnowflake returns a new snowflake
func GenSnowflake() (string, error) {
	sf, err := spaceflake.Generate(settings)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(sf.ID(), 10), nil
}
