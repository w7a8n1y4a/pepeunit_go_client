package pepeunit

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug    LogLevel = "Debug"
	LogLevelInfo     LogLevel = "Info"
	LogLevelWarning  LogLevel = "Warning"
	LogLevelError    LogLevel = "Error"
	LogLevelCritical LogLevel = "Critical"
)

// GetIntLevel returns the integer representation of the log level
func (l LogLevel) GetIntLevel() int {
	levelMapping := map[LogLevel]int{
		LogLevelDebug:    0,
		LogLevelInfo:     1,
		LogLevelWarning:  2,
		LogLevelError:    3,
		LogLevelCritical: 4,
	}
	return levelMapping[l]
}

// SearchTopicType represents the type of topic search
type SearchTopicType string

const (
	SearchTopicTypeUnitNodeUUID SearchTopicType = "unit_node_uuid"
	SearchTopicTypeFullName     SearchTopicType = "full_name"
)

// SearchScope represents the scope of topic search
type SearchScope string

const (
	SearchScopeAll    SearchScope = "all"
	SearchScopeInput  SearchScope = "input"
	SearchScopeOutput SearchScope = "output"
)

// DestinationTopicType represents the type of destination topic
type DestinationTopicType string

const (
	DestinationTopicTypeInputBaseTopic  DestinationTopicType = "input_base_topic"
	DestinationTopicTypeOutputBaseTopic DestinationTopicType = "output_base_topic"
	DestinationTopicTypeInputTopic      DestinationTopicType = "input_topic"
	DestinationTopicTypeOutputTopic     DestinationTopicType = "output_topic"
)

// BaseInputTopicType represents base input topic types
type BaseInputTopicType string

const (
	BaseInputTopicTypeUpdatePepeunit       BaseInputTopicType = "update/pepeunit"
	BaseInputTopicTypeEnvUpdatePepeunit    BaseInputTopicType = "env_update/pepeunit"
	BaseInputTopicTypeSchemaUpdatePepeunit BaseInputTopicType = "schema_update/pepeunit"
	BaseInputTopicTypeLogSyncPepeunit      BaseInputTopicType = "log_sync/pepeunit"
)

// BaseOutputTopicType represents base output topic types
type BaseOutputTopicType string

const (
	BaseOutputTopicTypeLogPepeunit   BaseOutputTopicType = "log/pepeunit"
	BaseOutputTopicTypeStatePepeunit BaseOutputTopicType = "state/pepeunit"
)

// RestartMode represents the restart mode for device updates
type RestartMode string

const (
	RestartModeRestartPopen  RestartMode = "restart_popen"
	RestartModeRestartExec   RestartMode = "restart_exec"
	RestartModeEnvSchemaOnly RestartMode = "env_schema_only"
	RestartModeNoRestart     RestartMode = "no_restart"
)
