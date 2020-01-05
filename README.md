# LDAP Query
LDAP-Query is a service which provides a REST API gateway to make queries against an LDAP directory.  It provides a read-only interface, changes to the directory are not supported.

It has specifically been tested against Active Directory, but there shouldn't be anything which is specific to AD so there is no obvious reason why it wouldn't work against other LDAP directories.

## Usage
The main configuration is done in the config file.  See that file, `config.toml`, for an explanation about how to configure it.  After changing the config file, the application will need to be restarted.

Other configuration is done via flags.

| Flag        | Description                              | Default Value |
| ----------- | ---------------------------------------- | ------------- |
| bind_port   | Port to bind the API endpoint to         | 9999          |
| version     | Display the application version and quit | false         |
| debug       | Enable debug logging                     | false         |

Once running you can run any query you want by sending a `POST` request to the `/search` endpoint with your query as the JSON payload.  Here is an example:

``` json
POST /search

{
    "filter": "(&(&(objectCategory=Person)(objectClass=User))(sn=skywalk*))",
    "scope": "sub",
    "base": "ou=xxx,dc=xxx,dc=xxx,dc=xx",
    "attributes": [
        "sAMAccountName",
        "cn",
        "givenName"
    ]
}
```

The `filter`, `base`, and `attributes` parameters are **required**.  The `scope` parameter is not required and will default to `base`.

No validation is carried out on the filter or attribute names, so if you don't get the results you expect make sure you ensure they are correct.

### Metrics
Application metrics are exported in [Prometheus](https://prometheus.io/) format to the `/metrics` endpoint.

### Logs
#### Windows
When running as a service logs are sent to the `Application` Event Log with a `Source` of `LDAP-Query`.

When running interactively, logs will be sent to `Stdout`.

#### Linux/Docker
Logs will be sent to `Stdout`.

## Running in production
It is best to setup `ldap-query` to run as a service.

### Windows
You can create a service from an executable using a few different methods:

- `sc.exe` - This is native to Windows and has the most support on more versions of Windows than any other option.  Difficult to do programmatically though should you want to.
- PowerShell using the `New-Service` cmdlet - Requires PowerShell 6 and above and makes doing things programmatically really easy.
- Something else such as [NSSM](https://nssm.cc/) - I'm not sure why you'd bother when `sc.exe` is available, but it's an option if you want it.

You'll probably want to ensure that the service is set to restart on failures, so check those settings.

When running as a service, logs will be sent to the `Application` Event Log using a `Source` of `LDAP-Query`.

When running interactively, logs will be sent to `Stdout`.

### Linux
Not sure, but likely just Docker?

**TODO** - Explain how to run as a service on Linux

### Docker
You can also run this as a container from https://hub.docker.com/r/rokett/ldap-query.  You will need to deal with where to store the config file though.

**TODO** - Document config file stuff for Docker

## How to setup local Dev environment
This project maintains dependencies under version control so building is really easy.

1. `go get github.com/rokett/ldap-query`.
2. Install [Packr v2](https://github.com/gobuffalo/packr/tree/master/v2); `go get -u github.com/gobuffalo/packr/v2/packr2`.

You can now carry out development.

To build the executable, just run `make` from the root of the repository.

The first time you run the executable a template config file, `config.toml`, will be created alongside the executable.  Update the config file as needed, and then run the executable again.

## Dockerfile
This Dockerfile will create a container that will set the entrypoint as `/ldap-queryd` so you can just pass in the command line options mentioned above to the container without needing to call the executable
