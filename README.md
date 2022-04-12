# Go client for Splunk

This is a REST API client for Splunk Enterprise that aims to expose the API in a Go-idiomatic way.

The main goal is to create functions that provide simple and hopefully complete enough mappings for Search and other functionality that is most commonly used. Suggestions and patches are welcome.

## Current State

This module is work-in-progress, please do not expect a stable API just yet. Currently supported functionality comprises:

- Authentication using username/password or token
- Simple "oneshot" searches
- "Export" searches
- "Raw" GET/POST/etc. requests to specific paths as building blocks for other use-cases

## License

GNU Lesser General Public License, version 3

## Author

Hilko Bengen <<bengen@hilluzination.de>>
