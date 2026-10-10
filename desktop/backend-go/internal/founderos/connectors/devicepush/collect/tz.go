package collect

import "time"

// local is the timezone day keys and zone-less timestamps are read in: the
// Mac's own, the day the operator lived. Tests pin it.
var local = time.Local
