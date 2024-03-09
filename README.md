### Building

In the root directory, run `make build`

### Running in development mode

In the root directory, run the unix socket with `APISIX_LISTEN_ADDRESS=unix:/tmp/runner.sock APISIX_CONF_EXPIRE_TIME=3600 ./go-runner run`

Then in apisix's configuration file `config.yaml` add this:

```
ext-plugin:
  path_for_test: /tmp/runner.sock
```

and restart apisix `apisix restart`

### Testing

#### Enable both firetail request and response filtering via apisix API:

**NOTE** You will need to run your application at localhost (127.0.0.1) port 1980. If you wish to point it elsewhere, change the "nodes" parameter from example below.
 
```
curl http://127.0.0.1:9180/apisix/admin/routes/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -d '
{
  "uri": "/profile/alice/comment",
  "plugins": {
    "ext-plugin-pre-req": {
      "conf": [
        {"name":"firetail", "value":"{\"body\":\"\"}"}         
      ]
    },
    "ext-plugin-post-resp": {
      "conf": [
        {"name":"firetail", "value":"{\"body\":\"\"}"}      
      ]
    }
  },
  "upstream": {
    "type": "roundrobin",
      "nodes": {
        "127.0.0.1:1980": 1
    }
  }
}'
```
