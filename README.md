# go-auto-sftp-check-modify
实现功能：定时检测本地文件是否更新，若文件更新则自动上传（sftp）远程服务器上。

## web访问路径
http://ip:8090/name?path=path，示例如下：
```
配置文件：
{
      "switch": "on",
      "name": "test",
      "local_os": "Windows",
      "local_base_dir": "E:\\workspace\\go\\test_dir",
      "remote_os": "Linux",
      "remote_base_dir": "/home/valvrave/workspace/go",
      "remote_address": "",
      "user": "",
      "passwd": "",
      "save_project":"E:\\workspace\\go\\go-auto-sftp-check-modify\\src\\save\\test.txt"
}
此时访问test_dir目录的url即：http://ip:8090/test?path=test_dir
访问test_dir子目录child_dir即：http://ip:8090/test?path=test_dir/child_dir
```
