# Live Migration Service

## CLI

The CLI is the tool that interacts with the Orchestrator to list connections, get status details, and initiate all cutover actions. It supports standard HTTP or HTTPS (provide your own certs) for the client. Overall, the tool simplifies interactions with the Orchestrator/LMS and has clear documentation via help commands for CLI usage.

### To add to path

In your `~/.zshrc`, `~/.bashrc`, or equivalent:

```
LMS_DIR="<path to your molt repo>/bin/lms"
export PATH="$PATH:$LMS_DIR"
```

### To run

Depending on your system's architecture, you'll probably be using a CLI with a different name. Substitute `molt-lms-cli` with the relevant name below to run.

To get help with commands:

```
molt-lms-cli help
```

Without TLS:

```
molt-lms-cli <insert sub commands here>
molt-lms-cli  connections list
```

With TLS:

```
# Template
molt-lms-cli <insert sub commands here> --tls-client-cert <path to your cert> --tls-client-key <path to your key> --tls-ca-cert <path to your CA cert>

# Example
molt-lms-cli  connections list --tls-client-cert ./client.crt --tls-client-key ./client.key --tls-ca-cert ./LMSCA.crt

# Alternatively, via env
export CLI_TLS_CLIENT_CERT=./client.crt
export CLI_TLS_CLIENT_KEY=./client.key
export CLI_TLS_CA_CERT=./LMSCA.crt
molt-lms-cli  connections list
```
