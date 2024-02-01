package testutils

import (
	"fmt"
	"runtime"
)

func GetLocalStoreAddrs(
	dialect string, port string,
) (localStoreListenAddr string, localStoreCrdbAccessAddr string) {

	const darwinLocalhostEndpoint = "host.docker.internal"
	const linuxLocalhostEndpoint = "172.17.0.1"
	const defaultLocalStorageServerPort = "4040"

	if port == "" {
		port = defaultLocalStorageServerPort
	}

	// Resources:
	// https://stackoverflow.com/questions/48546124/what-is-linux-equivalent-of-host-docker-internal
	// https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host
	// In the CI, the databases are all spin up in docker-compose,
	// which not necessarily share the network with the host.
	// When importing the data to the target database,
	// it requires the database reaches the local storage server
	// (spun up on host network) from the container (i.e. from
	// the container's network). According to the 2 links
	// above, the `localhost` on the host network is accessible
	// via different endpoint based on the operating system:
	// - Linux, Windows: 172.17.0.1
	// - MacOS: host.docker.internal
	switch runtime.GOOS {
	case "darwin":
		localStoreListenAddr = fmt.Sprintf("localhost:%s", port)
		localStoreCrdbAccessAddr = fmt.Sprintf("%s:%s", darwinLocalhostEndpoint, port)
	default:
		switch dialect {
		case "crdb":
			// Here the target db is cockroachdbtartget in .github/docker-compose.yaml,
			// which cannot be spun up on the host network (with ` network_mode: host`).
			// The reason is the docker image for crdb only allows listen-addr to be
			// localhost:26257, which will conflict with the `cockroachdb` container.
			// We thus has to let cockroachdbtartget lives in its own network and port-forward.
			// In this case in Linux, the host's localhost can only be accessed via `172.17.0.1`
			// from the container's network.
			localStoreListenAddr = fmt.Sprintf("%s:%s", linuxLocalhostEndpoint, port)
		case "pg", "mysql":
			// Here the target db is cockroachdb in .github/docker-compose.yaml,
			// which is directly spun up on the host network (with ` network_mode: host`).
			// In Linux case, it can directly access the host server via localhost.
			localStoreListenAddr = fmt.Sprintf("localhost:%s", port)
		default:
			panic(fmt.Sprintf("unknown dialect: %s", dialect))
		}
		localStoreCrdbAccessAddr = localStoreListenAddr
	}
	return localStoreListenAddr, localStoreCrdbAccessAddr
}
