# LDAP Query
LDAP-Query is a service which provides a REST API gateway to make queries against an LDAP directory.  It provides a read-only interface; changes to the directory are not supported.

It has specifically been tested against Active Directory, but there shouldn't be anything which is specific to AD so there is no obvious reason why it wouldn't work against other LDAP directories.

## Usage
Configuration is passed via flags.  If you need to query multiple domains, just run multiple instances of the application; remember to change the port that the application binds to.

| Flag              | Description                                                                                                  | Default Value |
| ----------------- | ------------------------------------------------------------------------------------------------------------ | ------------- |
| port              | Port the application listens on                                                                              | 9999          |
| allowed_sources   | Comma separated list of IPs to accept requests from                                                          | none          |
| directory_hosts   | Comma separated list of LDAP hosts to query; these should all be in the same domain                          | 9280          |
| directory_bind_dn | Full distinguished name of user account used to bind to the directory                                        | none          |
| directory_bind_pw | Password for the user account used to bind to the directory.  This does NOT need to be a privileged account. | none          |
| version           | Display application version information                                                                      | false         |
| service           | Manage Windows services; install, uninstall, start, and stop                                                 | none          |
| help              | Display help                                                                                                 | false         |
| debug             | Enable debug logging                                                                                         | false         |

````
ldap-queryd.exe --allowed_sources "172.16.124.34" --directory_hosts "192.168.1.22,192.168.1.56" --directory_bind_dn "CN=account1,CN=Users,DC=my,DC=domain" --directory_bind_pw "complex_password"
````

A machine with IP 172.16.124.34 is allowed to send request to LDAP hosts at 192.168.1.22 and 192.168.1.56 on the default port of 389, using the account details provided.

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

No validation is carried out on the filter or attribute names, so if you don't get the results you expect make sure you check that they are correct.

:warning: If the object you are searching for has brackets in the name, either `(` or `)`, you will need to escape the filter.  So a filter like `(&(cn=my group (admins),dc=xxx,dc=xxx,dc=xxx)(objectCategory=group))` needs to be like this -> `(&(cn=my group \\28admins\\29,dc=xxx,dc=xxx,dc=xxx)(objectCategory=group))`.

To display the application version run the application with the `--version` flag.

### Metrics
Application metrics are exported in [Prometheus](https://prometheus.io/) format to the `/metrics` endpoint.

### Logs
#### Windows
When running as a service logs are sent to the `Application` Event Log with a `Source` of `LDAP-Query`.

When running interactively, logs will be sent to `Stdout`.

#### Linux/Docker
Logs will be sent to `Stdout`.

## Running in production
It is best to setup `ldap-query` to run as a service/daemon.

### Windows
You can install the service like so:

```
ldap-queryd.exe --service install
```

Uninstall like so:

```
ldap-queryd.exe --service uninstall
```

You'll probably want to ensure that the service is set to restart on failures, so check those settings.

When running as a service, logs will be sent to the `Application` Event Log using a `Source` of `LDAP-Query`.

When running interactively, logs will be sent to `Stdout`.

### Linux
Not sure, but likely just Docker?

**TODO** - Explain how to run as a service on Linux

### Docker
You can also run this as a container by pulling the image from https://hub.docker.com/r/rokett/ldap-query.  You will need to deal with where to store the config file though.

**TODO** - Document config file stuff for Docker

## How to setup local Dev environment
This project maintains dependencies under version control, using Go modules, so building is really easy.

1. `go get github.com/rokett/ldap-query`.

You can now carry out development.

To build the executable, just run `make` from the root of the repository.

## Dockerfile
This Dockerfile will create a container that will set the entrypoint as `/ldap-queryd` so you can just pass in the command line options mentioned above to the container without needing to call the executable
