// Copyright 2024 Cockroach Labs Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/cockroachdb/errors"
)

type serviceName string

const (
	mysqlServiceName           serviceName = "mysql"
	pgServiceName              serviceName = "postgresql"
	cockroachServiceName       serviceName = "cockroachdb"
	cockroachTargetServiceName serviceName = "cockroachdbtarget"
)

func getContainerNameFromService(k serviceName) string {
	return fmt.Sprintf("github-%s-1", k)
}

var driverDialectToServices = map[string][2]serviceName{
	"mysql": {mysqlServiceName, cockroachServiceName},
	"pg":    {pgServiceName, cockroachServiceName},
	"crdb":  {cockroachServiceName, cockroachTargetServiceName},
}

const ymlPath = "../.github/docker-compose.yml"

// Setup is to start the required database containers based on the driver dialect.
func Setup(driverDialect string) error {
	_, err := os.ReadFile(ymlPath)
	if err != nil {
		return errors.Wrapf(err, "cannot read from yml")
	}
	services, ok := driverDialectToServices[driverDialect]
	if !ok {
		return errors.AssertionFailedf("unknown driver dialect: %s", driverDialect)
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker compose --file %s up -d %s %s", ymlPath, services[0], services[1]))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func TearDown() error {
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker compose --file %s down", ymlPath))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func confirmDBUp(s serviceName) error {
	// The commands to check the db accessibility are hard-coded based on the
	// network settings in .github/docker-compose.yml.
	serviceNameToCmd := map[serviceName]string{
		cockroachServiceName:       "psql 'postgres://root@localhost:26257/defaultdb?sslmode=disable' -c \"SELECT 1\"",
		cockroachTargetServiceName: "psql 'postgres://root@localhost:26258/defaultdb?sslmode=disable' -c \"SELECT 1\"",
		mysqlServiceName:           "mysql -u root  -h '0.0.0.0' -P 3306 --database=defaultdb --execute=\"SELECT 1\"",
		pgServiceName:              "psql 'postgres://postgres@localhost:5432/defaultdb' -c \"SELECT 1\"",
	}

	if cmd, ok := serviceNameToCmd[s]; ok {
		execCmd := exec.Command("/bin/sh", "-c", cmd)

		var stderr bytes.Buffer
		execCmd.Stderr = &stderr
		if err := execCmd.Run(); err != nil {
			if stderrMsg := stderr.String(); stderrMsg != "" {
				return errors.Newf(stderrMsg)
			}
			return err
		}
		return nil
	}
	return errors.AssertionFailedf("unknown service name: %s", s)
}

func ConfirmContainersRunning(t *testing.T, driverDialect string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 5 * time.Minute
	b.MaxInterval = 2 * time.Second

	serviceNames, ok := driverDialectToServices[driverDialect]
	if !ok {
		return errors.AssertionFailedf("unknown driver dialect: %s", driverDialect)
	}

	for _, s := range serviceNames {
		contName := getContainerNameFromService(s)

		checkDBUpF := func() error {
			dbUpErr := confirmDBUp(s)
			if dbUpErr != nil {
				t.Logf("database for %q is queriable with error: %s\nretrying...", contName, dbUpErr)
				return dbUpErr
			}
			return nil
		}

		checkContUpF := func() error {
			cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker container inspect -f '{{.State.Running}}' %s", contName))
			contUpRes, contUpErr := cmd.Output()
			if contUpErr != nil {
				t.Logf("container %q is not fully started with error: %s\nretrying...", contName, contUpErr)
				return contUpErr
			}
			contUpResStr := strings.TrimSpace(string(contUpRes))
			switch contUpResStr {
			case "true":
				if dbUpErr := backoff.Retry(checkDBUpF, b); dbUpErr != nil {
					return errors.Wrapf(dbUpErr, "container %q is up but db is not yet accessible", contName)
				}
				return nil
			default:
				return errors.Newf("container %q is not accessible", contName)
			}
		}

		if checkContUpErr := backoff.Retry(checkContUpF, b); checkContUpErr != nil {
			return checkContUpErr
		}
	}
	return nil
}
