package stock

import "time"

// timeNow is extracted for testability.
var timeNow = func() time.Time { return time.Now().UTC() }
