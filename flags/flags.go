// flags defines flags used by primary, backup, and client.
package flags

import "flag"

var Mode = flag.String("mode", "", "'server', 'client', or 'backup' mode")
var Port = flag.Int("port", 0, "used as the port number in 'server' and 'backup' mode. Used as the primary port to connect to in 'client' mode")
var Id = flag.Int("id", 0, "ID of the server, backup, or client.")
