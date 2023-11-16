# CLI Manifest Generator

This Go program generates an HTML page with the relevant versions and download links. It pulls from the versions file that exists at https://molt.cockroachdb.com/molt/cli/versions.txt or defaults to base-versions.txt.

In order to use:

```
go run . --version-file "base-versions.txt" --title "Versions"
```
