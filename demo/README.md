# LMS Demo

This directory contains everything required to bring up an end to end demo of the LMS inside of Docker. The docker-compose file will create the LMS, Orchestrator, CRDB, MySQL, CDCSink, Workload, Prometheus, and Grafana. All of these elements together will serve to create a complete end to end demo experience

# Setup

To run the demo, run the following command.

    docker-compose pull && docker-compose up --scale lms=3 -d

This will pull all images to ensure they are up to date and then bring up all the resources.

**Important**: Ensure that your LMS CLI version is the latest binary as well so that you can interact with the LMS pods.
**Important**: Ensure that you do not have any local processes listening on ports 26257, 3306, 3000, and 9046-9048.

You will also want to export the following or add it to your terminal profile.

    export CLI_ORCHESTRATOR_URL=http://localhost:4200

# Interacting with the components
The following components you can interact with by using the respective commands.

 1. The LMS nodes can be accessed with the respective mysql command. You can change the port number to 9047 or 9048 depending on which instance you would like to access.
 `mysql -u root -p'password'  -h '127.0.0.1' -P 9046 -D defaultdb`
 
 2. The MySql db can be accessed with the following command.
 `mysql -u root -p'password'  -h '127.0.0.1' -P 3306 -D defaultdb`
 
 3. The CRDB instance can be accessed with the following command.
 `psql -U root -h'127.0.0.1' -p 26257 -d defaultdb`
 
 4. Grafana can be accessed on **localhost:3000** and the login is admin/admin

# How the demo works
The demo will start as soon as you run the docker-compose up command. The workload too will start generating load to the LMS instances as soon as the database resources are ready. There may be a few container restarts as everything comes online.

The LMS nodes are set to shadow mode none so all traffic will only be going to the MySQL instances. CDCSink will be replicating all changes from MySQL to CRDB. When ready, you can use the CLI to initiate a consistent cutover. You can then tail the CDCSink logs and wait until there are no more logs updating indicating that replication is up to date. You could also do a count of the rows on the source and target db. Once replication has caught up, you can then commit the cutover.

# Teardown
When finished with the demo, simply run

    docker-compose down -v
This will tear down all the resources brought up and delete the temporary volumes created so the next time you run the demo it starts from a clean slate.
