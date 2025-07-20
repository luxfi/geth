// Package common provides formatting utilities
package common

import (
	"github.com/ethereum/go-ethereum/common"
)

// PrettyDuration is a wrapper for time.Duration for pretty printing
type PrettyDuration = common.PrettyDuration

// PrettyAge is a wrapper for time.Time for pretty printing
type PrettyAge = common.PrettyAge

// PrettyBytes is a wrapper for byte sizes for pretty printing
type PrettyBytes = common.PrettyBytes
