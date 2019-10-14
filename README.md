# go-utils

## app.ini 示例
```ini
;debug、test、release
mode = debug
host =
port = 8080

[log]
path = {$LOG_DIR}/logs

[gorm]
; true 打开，false 关闭，""只记录错误日志
log.mode =

[database]
test.drive = mysql
test.host = 127.0.0.1
test.port = 3306
test.username = root
test.password =
test.db = test
test.charset = utf8

pg.drive = postgresql
pg.host = 127.0.0.1
pg.port = 5432
pg.username = root
pg.password =
pg.db = test
pg.sslmode = 

[redis]
test.host = 127.0.0.1
test.port = 6379
test.auth =

```