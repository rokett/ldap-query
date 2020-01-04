REST API gateway for LDAP, specifically Active Directory.

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


# TO build
go get -u github.com/gobuffalo/packr/v2/packr2
