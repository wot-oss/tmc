package repos

import "time"

const (
	indexLockTimeout    = 5 * time.Second
	indexLockRetryDelay = 13 * time.Millisecond
)

//fs
const (
	defaultDirPermissions  = 0775
	defaultFilePermissions = 0664
	TMExt                  = ".tm.json"
)

//s3
const (
	lockLeaseTime       = 60 * time.Second
	lockRenewalInterval = 30 * time.Second
)
