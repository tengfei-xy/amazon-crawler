# amazon-crawler

一种在亚马逊平台分布式抓取TRN(Trade Register Number)的工具，适用多种主流操作系统

# 一、项目说明

本项目是通过关键词在亚马逊进行搜索的，并将搜索结果保留到数据库中并提取每个商品的卖家的信息，最后从卖家信息中提取TRN。由于每个步骤相互独立，因此适合分布式运行。

# 二、初始化说明

[下载](https://github.com/tengfei-xy/amazon-crawler/releases)

## 数据库

仅支持mysql，DDL代码参考[文件](https://github.com/tengfei-xy/amazon-crawler/blob/main/sql/ddl.sql)，

导入文件并初始化数据库参考命令：

```
bin/mysql -u root -p < xxx.sql
```

由于工具需要关键词来进行搜索，所以需要初始化一些关键词，参考[关键词数据库文件]((https://github.com/tengfei-xy/amazon-crawler/blob/main/sql/category.sql))

导入方式

```
bin/mysql -D amazon -u root -p < xxx.sql
```



## 配置文件

复制config.yaml.save为config.yaml

配置文件具体含义参考[文件中的注释](https://github.com/tengfei-xy/amazon-crawler/blob/main/config.yaml.save)

## cookie

你应该为每个配置文件中的host_id填充cookie，填充位置是数据库的cookie表，分别包含host_id和对应的cookie，

cookie格式如下

```
session-id=xxxx; session-id-time=2082787201l; i18n-prefs=GBP; sp-cdn="L5Z9:JP"; ubid-acbuk=xxx; session-token="xxxx"; csm-hit=xxx
```

注：亚马逊的cokie，半小时左右后会失效，但也会传回有效cookie，接着用就行。替换三次左右，cookie就变成天单位的有效时长了。至少有个2天。

注：cookie在程序运行时可随时修改，每次发送http请求时都会从数据库从重新获取一次

注：对于从亚马逊网页中获取cookie时，最好同意页面中提示的cookie，让cookie的存活更久

# 三、启动

```
./amazon-crawler -c xxx.yaml
```



# 其他

**运行（4天）情况：**

1k多个关键词，两次搜索，共发现235334个产品

```sql
mysql> select count(*) as `计数` from product;
+--------+
| 计数   |
+--------+
| 235334 |
+--------+
1 row in set (0.06 sec)
```

其中，对每个产品检查商家信息

```sql
mysql> SELECT * FROM amazon.产品检查表;
+--------------+--------------+
| 状态         | 链接数量     |
+--------------+--------------+
| 未搜索       |        32691 |
| 准备检查     |          391 |
| 检查结束     |       100891 |
| 其他错误     |           13 |
| 没有商家     |       101348 |
+--------------+--------------+
5 rows in set (0.10 sec)
```

**问题反馈（微信号）：**SXL--LP



