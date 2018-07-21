# Log Agent hook for logrus

Works for [Logstash](https://www.elastic.co/products/logstash) or [Gogstash](https://github.com/tsaikd/gogstash).

## Usage

```go
package main

import (
    "github.com/tengattack/logrus-agent-hook"
    "github.com/sirupsen/logrus"
    "net"
)

func main() {
    log := logrus.New()
    conn, err := net.Dial("tcp", "logstash.mycompany.net:8911")
    if err != nil {
        log.Fatal(err)
    }
    hook := logrusagent.New(conn, logrusagent.DefaultFormatter(logrus.Fields{"app_id": "foo"}))

    log.Hooks.Add(hook)
    log.Info("Hello World!")
}
```

Then, it becomes:

``` json
{
  "@timestamp": "2018-07-21T14:34:42+09:00",
    "@version": 1,
      "app_id": "foo",
       "level": "INFO",
     "message": "Hello World!"
}
```

# Inspired

[logrus-logstash-hook](https://github.com/bshuster-repo/logrus-logstash-hook)

# License

MIT
