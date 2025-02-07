# template-service-go

Template repository for backend services written in Go

# How to run the service

1. Create configuration file by copying and pasting `etc/tpl/conf.template.json` to `etc/cfg/conf.json`

2. Generate the swagger by running

```shell
make swag-install
make swaggo
```
